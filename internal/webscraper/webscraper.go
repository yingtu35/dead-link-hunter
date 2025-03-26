package webscraper

type DeadLinkMsg struct {
	parentUrl string
	url       string
}

type Page struct {
	DeadLinkCount int
	DeadLinks     []string
}

type WebScraper interface {
	// StartHunting starts the hunting process
	StartHunting()

	// GetResults returns the results of the hunting process
	GetResults() *map[string]*Page

	// PrintResults prints the results of the hunting process
	PrintResults()
}
