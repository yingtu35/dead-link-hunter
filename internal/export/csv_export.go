package export

import (
	"log"
	"os"
	"strconv"

	"github.com/gocarina/gocsv"
	"github.com/yingtu35/dead-link-hunter/internal/webscraper"
)

type DeadLinkRow struct {
	Page      string `csv:"Page,omitempty"`
	Counts    string `csv:"Counts,omitempty"`
	DeadLinks string `csv:"Dead Links"`
}

type CSVExporter struct{}

func NewCSVExporter() Exporter {
	return &CSVExporter{}
}

func (e *CSVExporter) Export(data *map[string]*webscraper.Page, filename string) error {
	file, err := os.Create(filename + ".csv")
	if err != nil {
		log.Printf("Error creating file %s: %v", filename, err)
		return err
	}
	defer file.Close()

	var result []DeadLinkRow

	if err := e.transformData(data, &result); err != nil {
		log.Printf("Error transforming data: %v", err)
		return err
	}

	if err := gocsv.MarshalFile(&result, file); err != nil {
		log.Printf("Error exporting data to CSV: %v", err)
		return err
	}
	return nil
}

func (e *CSVExporter) transformData(data *map[string]*webscraper.Page, result *[]DeadLinkRow) error {
	for url, page := range *data {
		for i, deadLink := range page.DeadLinks {
			if i == 0 {
				*result = append(*result, DeadLinkRow{Page: url, Counts: strconv.Itoa(page.DeadLinkCount), DeadLinks: deadLink})
			} else {
				*result = append(*result, DeadLinkRow{DeadLinks: deadLink})
			}
		}
	}
	return nil
}
