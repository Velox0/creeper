package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"

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

// CrawlerState holds crawling state and statistics
type CrawlerState struct {
	visited      map[string]bool
	incoming     map[string]int
	maxPages     int
	pagesVisited int
	maxDepth     int
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
					absStr := absURL.String()
					if absStr != from {
						state.incoming[absStr]++
					}
					if !state.visited[absStr] && state.pagesVisited < state.maxPages {
						fmt.Println(absStr)
						state.visited[absStr] = true
						state.pagesVisited++
						doc, err := fetch(absStr)
						if err == nil {
							visitLinks(base, doc, state, absStr, depth+1)
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
	state := &CrawlerState{
		visited:      map[string]bool{startURL: true},
		incoming:     map[string]int{startURL: 0},
		maxPages:     *maxPages,
		pagesVisited: 1,
		maxDepth:     *maxDepth,
	}
	visitLinks(base, doc, state, startURL, 1)

	if *showSummary {
		fmt.Println("\nSummary of visited pages and incoming link counts:")
		for url, count := range state.incoming {
			fmt.Printf("%s : %d incoming links\n", url, count)
		}
	}
}
