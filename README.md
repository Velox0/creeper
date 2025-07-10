# Creeper Web Crawler

A simple Go-based web crawler that visits pages on a domain, prints a summary of incoming links, and can generate a sitemap XML.

## Usage

```
go run main.go [flags] <url>
```

### Flags
- `-n <num>`   : Maximum number of pages to visit (default: 100)
- `-i <num>`   : Maximum recursion depth (0 = unlimited)
- `-s`         : Show summary table (default: true)
- `-x`         : Generate sitemap XML (`sitemap.xml`)

### Example

```
go run main.go -n 50 -x https://example.com
```

This will crawl up to 50 pages on example.com and generate a `sitemap.xml` file. 