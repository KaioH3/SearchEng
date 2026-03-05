package engine

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// DuckDuckGo searches via HTML scraping of html.duckduckgo.com.
type DuckDuckGo struct {
	Client *http.Client
}

func (d *DuckDuckGo) Name() string { return "DuckDuckGo" }

func (d *DuckDuckGo) Search(query string, page int) ([]Result, error) {
	params := url.Values{}
	params.Set("q", query)
	if page > 1 {
		params.Set("s", fmt.Sprintf("%d", (page-1)*30))
	}

	req, err := http.NewRequest("GET", "https://html.duckduckgo.com/html/?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0")

	resp, err := d.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo: status %d", resp.StatusCode)
	}

	return d.parse(resp.Body)
}

func (d *DuckDuckGo) client() *http.Client {
	if d.Client != nil {
		return d.Client
	}
	return http.DefaultClient
}

func (d *DuckDuckGo) parse(r io.Reader) ([]Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var results []Result
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && hasClass(n, "result__a") {
			result := Result{Source: d.Name()}
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					result.URL = extractDDGURL(attr.Val)
				}
			}
			result.Title = textContent(n)

			// Look for snippet in sibling
			snippet := findNextClass(n.Parent, "result__snippet")
			if snippet != nil {
				result.Snippet = textContent(snippet)
			}

			if result.URL != "" && result.Title != "" {
				results = append(results, result)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return results, nil
}

// extractDDGURL extracts the actual URL from DuckDuckGo's redirect URL.
func extractDDGURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if uddg := u.Query().Get("uddg"); uddg != "" {
		return uddg
	}
	return rawURL
}

// hasClass checks if an HTML node has a specific CSS class.
func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// textContent returns the concatenated text content of a node.
func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return strings.TrimSpace(sb.String())
}

// findNextClass searches siblings and their children for a node with the given class.
func findNextClass(parent *html.Node, class string) *html.Node {
	if parent == nil {
		return nil
	}
	// Search within the parent's parent (the result container)
	container := parent.Parent
	if container == nil {
		container = parent
	}
	var found *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && hasClass(n, class) {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(container)
	return found
}
