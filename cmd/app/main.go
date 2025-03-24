package main

import (
	"flag"
	"log"
	"os"

	"github.com/yingtu35/dead-link-hunter/internal/webscraper"
)

func main() {
	log.SetFlags(0)

	url := flag.String("url", "https://scrape-me.dreamsofcode.io/", "URL to fetch")
	flag.Parse()

	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Get all dead links
	dlh := webscraper.NewDeadLinkHunter(*url)
	dlh.StartHunting()

	dlh.PrintResults()
}
