package webscraper

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/rodaine/table"
	"github.com/yingtu35/dead-link-hunter/pkg/domain"
	"golang.org/x/net/html"
)

type Page struct {
	deadLinkCount int
	deadLinks     []string
}

type DeadLinkHunter struct {
	url                string           // The URL to start the hunting
	protocol           string           // The protocol of the URL
	domain             string           // The domain of the URL
	visitedPages       map[string]bool  // A map to keep track of visited pages
	pagesWithDeadLinks map[string]*Page // A map to keep track of pages with dead links
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
	return &DeadLinkHunter{
		url:                url,
		protocol:           protocol,
		domain:             domain,
		visitedPages:       make(map[string]bool),
		pagesWithDeadLinks: make(map[string]*Page),
	}
}

func (d *DeadLinkHunter) StartHunting() {
	d.hunt("", d.url)
}

func (d *DeadLinkHunter) hunt(parentUrl, url string) {
	if d.visitedPages[url] {
		return
	}

	if !domain.IsSameDomain(d.domain, url) {
		return
	}

	// Mark the page as visited
	d.visitedPages[url] = true

	log.Printf("fetching page %s", url)
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching %s: %v", url, err)
		return
	}
	defer res.Body.Close()

	log.Printf("response status code: %d", res.StatusCode)
	if res.StatusCode > 299 {
		// * Dead link found, add it to the pagesWithDeadLinks map
		if _, ok := d.pagesWithDeadLinks[parentUrl]; parentUrl != "" && !ok {
			d.pagesWithDeadLinks[parentUrl] = &Page{
				deadLinkCount: 0,
				deadLinks:     []string{},
			}
		}
		d.pagesWithDeadLinks[parentUrl].deadLinkCount++
		d.pagesWithDeadLinks[parentUrl].deadLinks = append(d.pagesWithDeadLinks[parentUrl].deadLinks, url)
		return
	}

	links, err := d.getAllLinks(res.Body)
	if err != nil {
		log.Printf("Error parsing links from %s: %v", url, err)
		return
	}

	for _, link := range links {
		d.hunt(url, link)
	}
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
