# Dead Link Hunter
A Web Scraping application written in Go that helps you find all dead links in your website

## Features
- **Concurrently** scan all pages of a website for dead links
- Handle **dynamic content scraping** with headless browsers
- Customizable scan depth
- Customizable concurrency level
- Export the results to a CSV file
- Export the results to a JSON file

## Usage
1. Clone the repository
```bash
git clone https://github.com/yingtu35/dead-link-hunter.git
cd dead-link-hunter
```

2. Run the server directly or build the binary
```bash
go run cmd/app/main.go --url yourwebsite.com
```
or
```bash
go build -o dead-link-hunter cmd/app/main.go
./dead-link-hunter --url yourwebsite.com
```

## Command-line Options

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--url` | Website URL to scan for dead links | - | Yes |
| `--static` | Enable static mode (faster but doesn't render JavaScript) | `false` | No |
| `--export` | Export format (`csv` or `json`) | - | No |
| `--filename` | Name of the export file (without extension) | `result` | No |
| `--maxDepth` | Maximum crawl depth from starting URL | 5 | No |
| `--maxConcurrency` | Maximum number of concurrent requests | 20 | No |
| `--timeout` | Request timeout in seconds | 10 | No |

### Examples

```bash
# Basic usage with default settings
./dead-link-hunter --url example.com

# Static scan with custom concurrency and export to CSV
./dead-link-hunter --url example.com --static --maxConcurrency 20 --export csv

# Deep scan with longer timeout and JSON export
./dead-link-hunter --url example.com --maxDepth 10 --timeout 20 --export json --filename deep-scan
```

## Roadmap
- [X] Support for JavaScript rendering with headless browsers
- [X] Add support for custom scan depth
- [X] Add support for custom concurrency level
- [X] Add support for exporting results

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.