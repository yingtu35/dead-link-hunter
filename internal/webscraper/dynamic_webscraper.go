package webscraper

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/rodaine/table"
	"github.com/yingtu35/dead-link-hunter/pkg/domain"
	"golang.org/x/sync/singleflight"
)

type DynamicHunter struct {
	pwClient           *playwright.Playwright // The Playwright client to use
	browser            *playwright.Browser    // The Playwright browser to use
	client             *http.Client           // The HTTP client to use
	url                string                 // The URL to start the hunting
	protocol           string                 // The protocol of the URL
	domain             string                 // The domain of the URL
	visitedPages       map[string]bool        // A map to keep track of visited pages
	deadUrls           map[string]bool        // A map to keep track of dead URLs
	pagesWithDeadLinks map[string]*Page       // A map to keep track of pages with dead links

	semaphore chan struct{} // A semaphore to limit the number of concurrent requests

	visitedMu sync.Mutex // A mutex to protect visitedPages and deadUrls
	pageMu    sync.Mutex // A mutex to protect pagesWithDeadLinks

	flightGroup singleflight.Group // A singleflight group to avoid duplicate requests
}

func NewDynamicHunter(url string) WebScraper {
	protocol, err := domain.GetProtocol(url)
	if err != nil {
		log.Fatalf("Error getting protocol from URL: %v", err)
	}
	domain, err := domain.GetDomain(url)
	if err != nil {
		log.Fatalf("Error getting domain from URL: %v", err)
	}

	var pwClient *playwright.Playwright
	var browser *playwright.Browser
	var client *http.Client

	pwOptions := playwright.RunOptions{
		SkipInstallBrowsers: true,
	}

	pw, err := playwright.Run(&pwOptions)
	if err != nil {
		log.Fatalf("Error creating Playwright client: %v", err)
	}
	pwClient = pw

	browserInstance, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("Error launching Playwright browser: %v", err)
	}
	browser = &browserInstance

	client = &http.Client{
		Timeout: DefaultTimeout * time.Second,
	}

	semaphore := make(chan struct{}, maxConcurrency)

	return &DynamicHunter{
		pwClient:           pwClient,
		browser:            browser,
		client:             client,
		url:                url,
		protocol:           protocol,
		domain:             domain,
		visitedPages:       make(map[string]bool),
		deadUrls:           make(map[string]bool),
		pagesWithDeadLinks: make(map[string]*Page),
		semaphore:          semaphore,
	}
}

func (dh *DynamicHunter) StartHunting() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Use singleflight for the initial URL too
		val, err, _ := dh.flightGroup.Do(dh.url, func() (interface{}, error) {
			return dh.hunt(dh.url, &wg)
		})

		if err != nil {
			log.Printf("Error hunting starting URL %s: %v", dh.url, err)
			return
		}

		// Handle the result if needed
		_, ok := val.(bool)
		if !ok {
			log.Printf("Error type assertion for starting URL %s", dh.url)
			return
		}

	}()

	wg.Wait()

	dh.close()
}

func (dh *DynamicHunter) PrintResults() {
	log.Println()
	if len(dh.pagesWithDeadLinks) == 0 {
		log.Println("No dead links found")
		return
	}
	tbl := table.New("Page", "Counts", "Dead Links")
	for url, page := range dh.pagesWithDeadLinks {
		for i, deadLink := range page.deadLinks {
			if i == 0 {
				tbl.AddRow(url, page.deadLinkCount, deadLink)
			} else {
				tbl.AddRow("", "", deadLink)
			}
		}
	}
	tbl.Print()
}

func (dh *DynamicHunter) close() {
	if dh.pwClient != nil && dh.browser != nil {
		if err := (*dh.browser).Close(); err != nil {
			log.Fatalf("Error closing browser: %v", err)
		}
		if err := dh.pwClient.Stop(); err != nil {
			log.Fatalf("Error stopping Playwright client: %v", err)
		}
	}
}

func (dh *DynamicHunter) hunt(url string, wg *sync.WaitGroup) (bool, error) {
	dh.semaphore <- struct{}{}
	defer func() {
		<-dh.semaphore
	}()

	// Check if the URL has already been visited
	dh.visitedMu.Lock()
	if dh.visitedPages[url] {
		isDead := dh.deadUrls[url]
		dh.visitedMu.Unlock()
		return isDead, nil
	}
	dh.visitedPages[url] = true
	dh.visitedMu.Unlock()

	if !domain.IsSameDomain(dh.domain, url) {
		return false, nil
	}

	// Check if it's a binary file URL
	if domain.IsBinaryFileUrl(url) {
		// Make HEAD request to check if the URL is valid
		log.Printf("fetching binary file %s", url)
		resp, err := dh.client.Head(url)
		if err != nil {
			return false, err
		}
		if resp.StatusCode > 299 {
			dh.visitedMu.Lock()
			dh.deadUrls[url] = true
			dh.visitedMu.Unlock()
			return true, nil
		}
		return false, nil
	}

	// Create a new context and page
	context, err := (*dh.browser).NewContext()
	if err != nil {
		return false, err
	}
	defer context.Close()

	context.SetDefaultNavigationTimeout(float64((DefaultTimeout * time.Second).Milliseconds()))
	page, err := context.NewPage()
	if err != nil {
		return false, err
	}

	log.Printf("fetching dynamic page %s", url)
	resp, err := page.Goto(url)
	if err != nil {
		return false, err
	}

	if resp.Status() > 299 {
		dh.visitedMu.Lock()
		dh.deadUrls[url] = true
		dh.visitedMu.Unlock()
		return true, nil
	}

	links, err := page.Locator("a").All()
	if err != nil {
		return false, err
	}
	for _, link := range links {
		href, err := link.GetAttribute("href")
		if err != nil {
			return false, err
		}
		linkURL, err := dh.constructURL(href)
		if err != nil {
			continue
		}
		wg.Add(1)
		go func(linkURL string) {
			// Decrement the wait group counter when the function returns
			defer wg.Done()

			val, err, _ := dh.flightGroup.Do(linkURL, func() (interface{}, error) {
				return dh.hunt(linkURL, wg)
			})
			if err != nil {
				log.Printf("Error hunting %s: %v", linkURL, err)
				return
			}
			isDead, ok := val.(bool)
			if !ok {
				log.Printf("Error type assertion %s", linkURL)
				return
			}
			if isDead {
				// * Dead link found, add it to the pagesWithDeadLinks map
				dh.pageMu.Lock()
				dh.addDeadLink(DeadLinkMsg{url, linkURL})
				dh.pageMu.Unlock()
			}
		}(linkURL)
	}
	return false, nil
}

func (dh *DynamicHunter) addDeadLink(deadlink DeadLinkMsg) {
	parentUrl := deadlink.parentUrl
	url := deadlink.url
	if _, ok := dh.pagesWithDeadLinks[parentUrl]; parentUrl != "" && !ok {
		dh.pagesWithDeadLinks[parentUrl] = &Page{
			deadLinkCount: 0,
			deadLinks:     []string{},
		}
	}
	dh.pagesWithDeadLinks[parentUrl].deadLinkCount++
	dh.pagesWithDeadLinks[parentUrl].deadLinks = append(dh.pagesWithDeadLinks[parentUrl].deadLinks, url)
}

func (dh *DynamicHunter) constructURL(url string) (string, error) {
	// if empty string, return error
	if url == "" {
		return "", errors.New("empty string")
	}
	if strings.HasPrefix(url, "#") {
		return "", errors.New("anchor link")
	}
	// if start with /, then it's a relative path
	if strings.HasPrefix(url, "/") {
		return dh.protocol + "://" + dh.domain + url, nil
	}
	// if start with http or https, then it's an absolute path
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url, nil
	}
	return "", errors.New("invalid URL")
}
