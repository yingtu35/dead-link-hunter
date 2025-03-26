package export

import (
	"encoding/json"
	"log"
	"os"

	"github.com/yingtu35/dead-link-hunter/internal/webscraper"
)

type Record struct {
	Page      string   `json:"Page"`
	Counts    int      `json:"Counts"`
	DeadLinks []string `json:"Dead Links"`
}

type JsonExporter struct{}

func NewJsonExporter() Exporter {
	return &JsonExporter{}
}

func (e *JsonExporter) Export(data *map[string]*webscraper.Page, filename string) error {
	file, err := os.Create(filename + ".json")
	if err != nil {
		log.Printf("Error creating file %s: %v", filename, err)
		return err
	}

	var result []Record
	e.transformData(data, &result)

	resultJson, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Printf("Error marshalling data: %v", err)
		return err
	}

	if _, err := file.Write(resultJson); err != nil {
		log.Printf("Error exporting data to JSON: %v", err)
		return err
	}
	return nil
}

func (e *JsonExporter) transformData(data *map[string]*webscraper.Page, result *[]Record) {
	for url, page := range *data {
		*result = append(*result, Record{
			Page:      url,
			Counts:    page.DeadLinkCount,
			DeadLinks: page.DeadLinks,
		})
	}
}
