package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"encoding/xml"

	"golang.org/x/net/html"
	"golang.org/x/term"
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
	if norm.Path == "" {
		norm.Path = "/"
	}
	return norm.String()
}

// pathOnly returns just the path from a URL string
func pathOnly(u *url.URL) string {
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	return p
}

type XMLUrlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []XMLUrl `xml:"url"`
}
type XMLUrl struct {
	Loc      string `xml:"loc"`
	Priority string `xml:"priority"`
}

// CrawlerState holds crawling state and statistics
type CrawlerState struct {
	visited      map[string]bool
	incoming     map[string]int
	maxPages     int
	pagesVisited int
	maxDepth     int
	paths        map[string]string // normalized url -> path only
	outgoing     map[string]int    // normalized url -> outgoing link count
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
					state.outgoing[fromNorm]++
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

func printSummaryTable(state *CrawlerState, startPath string) {
	// Get terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 30 {
		width = 80
	}
	countCol := 6
	pathCol := width - countCol - 3

	paths := make([]string, 0, len(state.paths))
	for _, p := range state.paths {
		paths = append(paths, p)
	}
	uniquePaths := make(map[string]struct{})
	for _, p := range paths {
		uniquePaths[p] = struct{}{}
	}
	finalPaths := make([]string, 0, len(uniquePaths))
	for p := range uniquePaths {
		finalPaths = append(finalPaths, p)
	}
	startIdx := -1
	for i, p := range finalPaths {
		if p == startPath {
			startIdx = i
			break
		}
	}
	if startIdx > 0 {
		finalPaths[0], finalPaths[startIdx] = finalPaths[startIdx], finalPaths[0]
	}
	if startIdx != 0 {
		sort.Strings(finalPaths[1:])
	} else {
		sort.Strings(finalPaths[1:])
	}

	// Header
	fmt.Printf("%-*s | %s\n", countCol, "Count", "Path")
	fmt.Println(strings.Repeat("-", width))

	for _, p := range finalPaths {
		// Find normalized url for this path
		var incoming int
		for norm, path := range state.paths {
			if path == p {
				incoming = state.incoming[norm]
				break
			}
		}
		// Split path into chunks
		pathRunes := []rune(p)
		for i := 0; i < len(pathRunes); i += pathCol {
			end := i + pathCol
			if end > len(pathRunes) {
				end = len(pathRunes)
			}
			chunk := string(pathRunes[i:end])
			if i == 0 {
				fmt.Printf("%-*d | %s\n", countCol, incoming, chunk)
			} else {
				fmt.Printf("%-*s | %s\n", countCol, "", chunk)
			}
		}
	}
}

func writeXML(state *CrawlerState) error {
	maxOutgoing := 1
	maxIncoming := 1
	for norm := range state.paths {
		if state.outgoing[norm] > maxOutgoing {
			maxOutgoing = state.outgoing[norm]
		}
		if state.incoming[norm] > maxIncoming {
			maxIncoming = state.incoming[norm]
		}
	}
	const outlinkWeight, inlinkWeight = 0.75, 0.25

	urls := make([]XMLUrl, 0, len(state.paths))
	for norm := range state.paths {
		out := state.outgoing[norm]
		in := state.incoming[norm]
		outNorm := float64(out) / float64(maxOutgoing)
		inNorm := float64(in) / float64(maxIncoming)
		priority := outlinkWeight*outNorm + inlinkWeight*inNorm
		urls = append(urls, XMLUrl{
			Loc:      norm,
			Priority: fmt.Sprintf("%.2f", priority),
		})
	}
	// Sort urls by priority descending
	sort.Slice(urls, func(i, j int) bool {
		return urls[i].Priority > urls[j].Priority
	})
	urlset := XMLUrlSet{Urls: urls}
	f, err := os.Create("sitemap.xml")
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	f.WriteString("<!-- Generated by github.com/velox0/creeper -->\n")
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	return enc.Encode(urlset)
}

func main() {
	maxPages := flag.Int("n", 100, "Maximum number of pages to visit")
	maxDepth := flag.Int("i", 0, "Maximum recursion depth (0 = unlimited)")
	showSummary := flag.Bool("s", true, "Show summary of incoming links at the end")
	xmlOut := flag.Bool("x", false, "Generate XML output (output.xml)")
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
	startPath := pathOnly(base)
	state := &CrawlerState{
		visited:      map[string]bool{normStart: true},
		incoming:     map[string]int{normStart: 0},
		maxPages:     *maxPages,
		pagesVisited: 1,
		maxDepth:     *maxDepth,
		paths:        map[string]string{normStart: startPath},
		outgoing:     map[string]int{normStart: 0},
	}
	visitLinks(base, doc, state, normStart, 1)

	if *showSummary {
		fmt.Println("\nSummary of visited pages and incoming link counts:")
		printSummaryTable(state, startPath)
	}
	if *xmlOut {
		err := writeXML(state)
		if err != nil {
			fmt.Println("Error writing XML:", err)
		} else {
			fmt.Println("XML written to sitemap.xml")
		}
	}
}
