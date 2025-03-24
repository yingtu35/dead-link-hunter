package webscraper

import (
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/rodaine/table"
	"github.com/yingtu35/dead-link-hunter/pkg/domain"
	"golang.org/x/net/html"
)

var visitedChBuffer = 100  // buffer channel size for visitedCh
var deadLinkChBuffer = 100 // buffer channel size for deadLinkCh

var maxConcurrency = 20 // maximum number of concurrent requests

type DeadLinkMsg struct {
	parentUrl string
	url       string
}

type Page struct {
	deadLinkCount int
	deadLinks     []string
}

type DeadLinkHunter struct {
	client             *http.Client     // The HTTP client to use
	url                string           // The URL to start the hunting
	protocol           string           // The protocol of the URL
	domain             string           // The domain of the URL
	visitedPages       map[string]bool  // A map to keep track of visited pages
	pagesWithDeadLinks map[string]*Page // A map to keep track of pages with dead links

	semaphore  chan struct{}    // A semaphore to limit the number of concurrent requests
	visitedCh  chan string      // A channel to keep track of visited pages
	deadLinkCh chan DeadLinkMsg // A channel to keep track of dead links
	doneCh     chan struct{}    // A channel to signal the hunting is done

	visitedMu sync.RWMutex // A mutex to protect visitedPages
}

func NewDeadLinkHunter(url string) *DeadLinkHunter {
	protocol, err := domain.GetProtocol(url)
	if err != nil {
		log.Fatalf("Error getting protocol from URL: %v", err)
	}
	domain, err := domain.GetDomain(url)
	if err != nil {
		log.Fatalf("Error getting domain from URL: %v", err)
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	semaphore := make(chan struct{}, maxConcurrency)
	visitedCh := make(chan string, visitedChBuffer)
	deadLinkCh := make(chan DeadLinkMsg, deadLinkChBuffer)
	doneCh := make(chan struct{})

	visitedMu := sync.RWMutex{}
	return &DeadLinkHunter{
		client:             client,
		url:                url,
		protocol:           protocol,
		domain:             domain,
		visitedPages:       make(map[string]bool),
		pagesWithDeadLinks: make(map[string]*Page),
		semaphore:          semaphore,
		visitedCh:          visitedCh,
		deadLinkCh:         deadLinkCh,
		doneCh:             doneCh,
		visitedMu:          visitedMu,
	}
}

func (d *DeadLinkHunter) StartHunting() {
	var wg sync.WaitGroup

	go func() {
		d.processMessages(&wg)
		close(d.doneCh)
	}()

	wg.Add(1)
	go d.hunt("", d.url, &wg)

	wg.Wait()

	close(d.visitedCh)
	close(d.deadLinkCh)
	<-d.doneCh
}

func (d *DeadLinkHunter) hunt(parentUrl, url string, wg *sync.WaitGroup) {
	// Decrement the wait group counter when the function returns
	defer wg.Done()

	// Acquire the semaphore
	d.semaphore <- struct{}{}
	defer func() {
		<-d.semaphore
	}()

	// Check if the page has been visited
	d.visitedMu.RLock()
	if d.visitedPages[url] {
		d.visitedMu.RUnlock()
		return
	}
	d.visitedMu.RUnlock()

	if !domain.IsSameDomain(d.domain, url) {
		return
	}

	// Mark the page as visited
	d.visitedCh <- url

	log.Printf("fetching page %s", url)
	res, err := d.client.Get(url)
	if err != nil {
		log.Printf("Error fetching %s: %v", url, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		// * Dead link found, add it to the pagesWithDeadLinks map
		d.deadLinkCh <- DeadLinkMsg{parentUrl, url}
		return
	}

	links, err := d.getAllLinks(res.Body)
	if err != nil {
		log.Printf("Error parsing links from %s: %v", url, err)
		return
	}

	for _, link := range links {
		wg.Add(1)
		go d.hunt(url, link, wg)
	}
}

func (d *DeadLinkHunter) processMessages(wg *sync.WaitGroup) {
	for {
		select {
		case url, ok := <-d.visitedCh:
			if !ok {
				d.visitedCh = nil
			} else {
				d.visitedMu.Lock()
				d.visitedPages[url] = true
				d.visitedMu.Unlock()
			}
		case deadLink, ok := <-d.deadLinkCh:
			if !ok {
				d.deadLinkCh = nil
			} else {
				d.addDeadLink(deadLink)
			}
		}

		// Check if both channels are closed
		if d.visitedCh == nil && d.deadLinkCh == nil {
			return
		}
	}
}

func (d *DeadLinkHunter) addDeadLink(deadlink DeadLinkMsg) {
	parentUrl := deadlink.parentUrl
	url := deadlink.url
	if _, ok := d.pagesWithDeadLinks[parentUrl]; parentUrl != "" && !ok {
		d.pagesWithDeadLinks[parentUrl] = &Page{
			deadLinkCount: 0,
			deadLinks:     []string{},
		}
	}
	d.pagesWithDeadLinks[parentUrl].deadLinkCount++
	d.pagesWithDeadLinks[parentUrl].deadLinks = append(d.pagesWithDeadLinks[parentUrl].deadLinks, url)
}

func (d *DeadLinkHunter) PrintResults() {
	tbl := table.New("Page", "Counts", "Dead Links")
	for url, page := range d.pagesWithDeadLinks {
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

func (d *DeadLinkHunter) getAllLinks(body io.Reader) ([]string, error) {
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

func (d *DeadLinkHunter) constructURL(url string) (string, error) {
	// if empty string, return empty string
	if url == "" {
		return "", errors.New("empty string")
	}
	// if start with /, then it's a relative path
	if url[0] == '/' {
		return d.protocol + "://" + d.domain + url, nil
	}
	return url, nil
}
