package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// fetches the HTML document from the given URL
func fetch(url string) (*html.Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return html.Parse(resp.Body)
}

// normalizeURL removes query params and fragments, returns scheme://host/path
func normalizeURL(u *url.URL) string {
	norm := *u
	norm.RawQuery = ""
	norm.Fragment = ""
	// Always keep trailing slash for root
	if norm.Path == "" {
		norm.Path = "/"
	}
	return norm.String()
}

// pathOnly returns just the path (and query if needed) from a URL string
func pathOnly(u *url.URL) string {
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	return p
}

// CrawlerState holds crawling state and statistics
type CrawlerState struct {
	visited      map[string]bool
	incoming     map[string]int
	maxPages     int
	pagesVisited int
	maxDepth     int
	paths        map[string]string // normalized url -> path only
}

// visitLinks finds all in-domain links and recursively visits them
func visitLinks(base *url.URL, n *html.Node, state *CrawlerState, from string, depth int) {
	if state.pagesVisited >= state.maxPages || (state.maxDepth > 0 && depth > state.maxDepth) {
		return
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				href := attr.Val
				u, err := url.Parse(href)
				if err != nil {
					continue
				}
				absURL := base.ResolveReference(u)
				if absURL.Host == base.Host && (absURL.Scheme == "http" || absURL.Scheme == "https") {
					normStr := normalizeURL(absURL)
					fromNorm := from
					if fromNorm != normStr {
						state.incoming[normStr]++
					}
					state.paths[normStr] = pathOnly(absURL)
					if !state.visited[normStr] && state.pagesVisited < state.maxPages {
						fmt.Println(normStr)
						state.visited[normStr] = true
						state.pagesVisited++
						doc, err := fetch(normStr)
						if err == nil {
							visitLinks(base, doc, state, normStr, depth+1)
						}
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		visitLinks(base, c, state, from, depth)
	}
}

func main() {
	maxPages := flag.Int("n", 100, "Maximum number of pages to visit")
	maxDepth := flag.Int("i", 0, "Maximum recursion depth (0 = unlimited)")
	showSummary := flag.Bool("s", true, "Show summary of incoming links at the end")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: go run main.go [flags] <url>")
		flag.PrintDefaults()
		return
	}
	startURL := flag.Arg(0)
	base, err := url.Parse(startURL)
	if err != nil {
		fmt.Println("Invalid URL:", err)
		return
	}
	doc, err := fetch(startURL)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return
	}
	normStart := normalizeURL(base)
	state := &CrawlerState{
		visited:      map[string]bool{normStart: true},
		incoming:     map[string]int{normStart: 0},
		maxPages:     *maxPages,
		pagesVisited: 1,
		maxDepth:     *maxDepth,
		paths:        map[string]string{normStart: pathOnly(base)},
	}
	visitLinks(base, doc, state, normStart, 1)

	if *showSummary {
		fmt.Println("\nSummary of visited pages and incoming link counts:")
		// Sort by path for pretty output
		paths := make([]string, 0, len(state.paths))
		for _, p := range state.paths {
			paths = append(paths, p)
		}
		// Remove duplicates
		uniquePaths := make(map[string]struct{})
		for _, p := range paths {
			uniquePaths[p] = struct{}{}
		}
		finalPaths := make([]string, 0, len(uniquePaths))
		for p := range uniquePaths {
			finalPaths = append(finalPaths, p)
		}
		// Sort
		sort.Strings(finalPaths)
		// Table header
		fmt.Printf("%-40s | %s\n", "Path", "Incoming Links")
		fmt.Println(strings.Repeat("-", 55))
		for _, p := range finalPaths {
			// Find normalized url for this path
			var incoming int
			for norm, path := range state.paths {
				if path == p {
					incoming = state.incoming[norm]
					break
				}
			}
			fmt.Printf("%-40s | %d\n", p, incoming)
		}
	}
}
