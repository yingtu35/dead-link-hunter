package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/yingtu35/dead-link-hunter/internal/export"
	"github.com/yingtu35/dead-link-hunter/internal/webscraper"
)

func main() {
	log.SetFlags(0)

	url := flag.String("url", "", "URL to fetch")
	static := flag.Bool("static", false, "Enable static scraping")
	exportType := flag.String("export", "", "Export file format (csv, json)")
	filename := flag.String("filename", "result", "Export file name")

	flag.Parse()

	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Get all dead links
	var dlh webscraper.WebScraper
	if *static {
		dlh = webscraper.NewStaticHunter(*url)
	} else {
		dlh = webscraper.NewDynamicHunter(*url)
	}
	start := time.Now()
	dlh.StartHunting()
	elapsed := time.Since(start)

	var exporter export.Exporter
	switch strings.ToLower(*exportType) {
	case "csv":
		exporter = export.NewCSVExporter()
		if err := exporter.Export(dlh.GetResults(), *filename); err != nil {
			log.Fatalf("Error exporting data: %v", err)
		}
	case "json":
		exporter = export.NewJsonExporter()
		if err := exporter.Export(dlh.GetResults(), *filename); err != nil {
			log.Fatalf("Error exporting data: %v", err)
		}
	default:
		// Print the results
		dlh.PrintResults()
	}

	log.Printf("Total Hunting Time: %s", elapsed)
}
