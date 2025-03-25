# Dead Link Hunter
A Web Scraping application written in Go that helps you find all dead links in your website

## Features
- **Concurrently** scan all pages of a website for dead links
- Handle **dynamic content scraping** with headless browsers
- Customizable scan depth (coming soon)
- Customizable concurrency level (coming soon)
- Export the results to a CSV file (coming soon)
- Export the results to a JSON file (coming soon)

## Usage
1. Clone the repository
```bash
git clone https://github.com/yingtu35/dead-link-hunter.git
cd dead-link-hunter
```

2. Run the server directly or build the binary
   Add the `--static` flag to scan static websites for improved performance
```bash
go run cmd/app/main.go --url yourwebsite.com (--static)
```
or
```bash
go build -o dead-link-hunter cmd/app/main.go
./dead-link-hunter --url yourwebsite.com (--static)
```

## Roadmap
- [X] Support for JavaScript rendering with headless browsers
- [ ] Add support for custom scan depth
- [ ] Add support for custom concurrency level
- [ ] Add support for exporting results

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.