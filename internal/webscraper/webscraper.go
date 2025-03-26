package webscraper

type DeadLinkMsg struct {
	parentUrl string
	url       string
}

type Page struct {
	DeadLinkCount int
	DeadLinks     []string
}

type ScraperOptions struct {
	MaxDepth       int
	MaxConcurrency int
	Timeout        int
}

type WebScraper interface {
	// SetHunterOptions sets the options for the hunter
	SetHunterOptions(options *ScraperOptions)

	// StartHunting starts the hunting process
	StartHunting()

	// GetResults returns the results of the hunting process
	GetResults() *map[string]*Page

	// PrintResults prints the results of the hunting process
	PrintResults()
}
