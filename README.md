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
- `-l`         : Crawl using localhost for the given domain (for local server testing; requests go to localhost, but Host header is set to the original domain)
- `-p <port>`  : Port to use with `-l` (default: 80). Only used if `-l` is set.

### Localhost Crawling

If you want to crawl a site served locally (e.g., for testing a production domain on your dev server), use the `-l` flag. This will make all requests to `http://localhost` (or the port you specify with `-p`), but set the `Host` header to the original domain. This is useful for testing how your site behaves as if it were live, but using your local server.

- When using `-l`, the crawler always uses the `http` protocol for requests to localhost, regardless of the original URL's scheme.

#### Example

```
go run main.go -l -p 8080 -n 20 example.com
```

This will crawl up to 20 pages, making requests to `http://localhost:8080`, but with the `Host` header set to `example.com`. 