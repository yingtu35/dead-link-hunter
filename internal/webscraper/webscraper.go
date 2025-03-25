package webscraper

type DeadLinkMsg struct {
	parentUrl string
	url       string
}

type Page struct {
	deadLinkCount int
	deadLinks     []string
}

type WebScraper interface {
	// StartHunting starts the hunting process
	StartHunting()

	// PrintResults prints the results of the hunting process
	PrintResults()
}
