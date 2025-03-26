package webscraper

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rodaine/table"
	"github.com/yingtu35/dead-link-hunter/pkg/domain"
	"golang.org/x/net/html"
	"golang.org/x/sync/singleflight"
)

type StaticHunter struct {
	scraperOptions     *ScraperOptions  // The scraper options to use
	client             *http.Client     // The HTTP client to use
	url                string           // The URL to start the hunting
	protocol           string           // The protocol of the URL
	domain             string           // The domain of the URL
	visitedPages       map[string]bool  // A map to keep track of visited pages
	deadUrls           map[string]bool  // A map to keep track of dead URLs
	pagesWithDeadLinks map[string]*Page // A map to keep track of pages with dead links

	semaphore chan struct{} // A semaphore to limit the number of concurrent requests

	visitedMu sync.Mutex // A mutex to protect visitedPages and deadUrls
	pageMu    sync.Mutex // A mutex to protect pagesWithDeadLinks

	flightGroup singleflight.Group // A singleflight group to avoid duplicate requests
}

func NewStaticHunter(url string) WebScraper {
	protocol, err := domain.GetProtocol(url)
	if err != nil {
		log.Fatalf("Error getting protocol from URL: %v", err)
	}
	domain, err := domain.GetDomain(url)
	if err != nil {
		log.Fatalf("Error getting domain from URL: %v", err)
	}

	client := &http.Client{
		Timeout: DefaultTimeout * time.Second,
	}

	semaphore := make(chan struct{})

	return &StaticHunter{
		scraperOptions:     &ScraperOptions{MaxDepth: MaxDepth, MaxConcurrency: MaxConcurrency, Timeout: DefaultTimeout},
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

func (d *StaticHunter) SetHunterOptions(options *ScraperOptions) {
	d.scraperOptions = options
	d.semaphore = make(chan struct{}, d.scraperOptions.MaxConcurrency)
	d.client.Timeout = time.Duration(d.scraperOptions.Timeout) * time.Second
}

func (d *StaticHunter) StartHunting() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Use singleflight for the initial URL too
		val, err, _ := d.flightGroup.Do(d.url, func() (interface{}, error) {
			return d.hunt(d.url, &wg, 0)
		})

		if err != nil {
			log.Printf("Error hunting starting URL %s: %v", d.url, err)
			return
		}

		// Handle the result if needed
		_, ok := val.(bool)
		if !ok {
			log.Printf("Error type assertion for starting URL %s", d.url)
			return
		}

	}()

	wg.Wait()
}

func (d *StaticHunter) GetResults() *map[string]*Page {
	return &d.pagesWithDeadLinks
}

func (d *StaticHunter) hunt(url string, wg *sync.WaitGroup, curDepth int) (bool, error) {
	// Acquire the semaphore
	d.semaphore <- struct{}{}
	defer func() {
		<-d.semaphore
	}()

	// Check if the URL has already been visited
	d.visitedMu.Lock()
	if d.visitedPages[url] {
		isDead := d.deadUrls[url]
		d.visitedMu.Unlock()
		return isDead, nil
	}
	d.visitedPages[url] = true
	d.visitedMu.Unlock()

	if !domain.IsSameDomain(d.domain, url) {
		return false, nil
	}

	// Check if it's a binary file URL
	if domain.IsBinaryFileUrl(url) {
		// Make HEAD request to check if the URL is valid
		log.Printf("fetching binary file %s", url)
		resp, err := d.client.Head(url)
		if err != nil {
			return false, err
		}
		if resp.StatusCode > 299 {
			d.visitedMu.Lock()
			d.deadUrls[url] = true
			d.visitedMu.Unlock()
			return true, nil
		}
		return false, nil
	}

	log.Printf("fetching page %s", url)
	res, err := d.client.Get(url)
	if err != nil {
		log.Printf("Error fetching %s: %v", url, err)
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		d.visitedMu.Lock()
		d.deadUrls[url] = true
		d.visitedMu.Unlock()
		return true, nil
	}

	// Check if the current depth is greater than the maximum depth
	if curDepth >= d.scraperOptions.MaxDepth {
		return false, nil
	}

	links, err := d.getAllLinks(res.Body)
	if err != nil {
		log.Printf("Error parsing links from %s: %v", url, err)
		return false, err
	}

	for _, link := range links {
		wg.Add(1)
		go func(link string) {
			// Decrement the wait group counter when the function returns
			defer wg.Done()

			val, err, _ := d.flightGroup.Do(link, func() (interface{}, error) {
				return d.hunt(link, wg, curDepth+1)
			})
			if err != nil {
				log.Printf("Error hunting %s: %v", link, err)
				return
			}
			isDead, ok := val.(bool)
			if !ok {
				log.Printf("Error type assertion %s", link)
				return
			}
			if isDead {
				// * Dead link found, add it to the pagesWithDeadLinks map
				d.pageMu.Lock()
				d.addDeadLink(DeadLinkMsg{url, link})
				d.pageMu.Unlock()
			}
		}(link)
	}
	return false, nil
}

func (d *StaticHunter) addDeadLink(deadlink DeadLinkMsg) {
	parentUrl := deadlink.parentUrl
	url := deadlink.url
	if _, ok := d.pagesWithDeadLinks[parentUrl]; parentUrl != "" && !ok {
		d.pagesWithDeadLinks[parentUrl] = &Page{
			DeadLinkCount: 0,
			DeadLinks:     []string{},
		}
	}
	d.pagesWithDeadLinks[parentUrl].DeadLinkCount++
	d.pagesWithDeadLinks[parentUrl].DeadLinks = append(d.pagesWithDeadLinks[parentUrl].DeadLinks, url)
}

func (d *StaticHunter) PrintResults() {
	log.Println()
	if len(d.pagesWithDeadLinks) == 0 {
		log.Println("No dead links found")
		return
	}

	tbl := table.New("Page", "Counts", "Dead Links")
	for url, page := range d.pagesWithDeadLinks {
		for i, deadLink := range page.DeadLinks {
			if i == 0 {
				tbl.AddRow(url, page.DeadLinkCount, deadLink)
			} else {
				tbl.AddRow("", "", deadLink)
			}
		}
	}
	tbl.Print()
}

func (d *StaticHunter) getAllLinks(body io.Reader) ([]string, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	var links []string
	for n := range doc.Descendants() {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					link, err := d.constructURL(a.Val)
					if err != nil {
						continue
					}
					links = append(links, link)
				}
			}
		}
	}

	return links, nil
}

func (d *StaticHunter) constructURL(url string) (string, error) {
	// if empty string, return error
	if url == "" {
		return "", errors.New("empty string")
	}
	if strings.HasPrefix(url, "#") {
		return "", errors.New("anchor link")
	}
	// if start with /, then it's a relative path
	if strings.HasPrefix(url, "/") {
		return d.protocol + "://" + d.domain + url, nil
	}
	// if start with http or https, then it's an absolute path
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url, nil
	}
	return "", errors.New("invalid URL")
}
