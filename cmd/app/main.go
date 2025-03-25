package main

import (
	"flag"
	"log"
	"os"
	"time"

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
	start := time.Now()
	dlh.StartHunting()
	elapsed := time.Since(start)

	dlh.PrintResults()
	log.Printf("Total Hunting Time: %s", elapsed)
}
