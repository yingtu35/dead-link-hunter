package export

import "github.com/yingtu35/dead-link-hunter/internal/webscraper"

type Exporter interface {
	// Export exports the data to the specified file
	Export(data *map[string]*webscraper.Page, filename string) error
}
