package engine

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Google searches via HTML scraping of google.com.
type Google struct {
	Client *http.Client
}

func (g *Google) Name() string { return "Google" }

func (g *Google) Search(query string, page int) ([]Result, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("hl", "en")
	if page > 1 {
		params.Set("start", fmt.Sprintf("%d", (page-1)*10))
	}

	req, err := http.NewRequest("GET", "https://www.google.com/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	setBrowserHeaders(req)

	resp, err := g.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google: status %d", resp.StatusCode)
	}

	return g.parse(resp.Body)
}

func (g *Google) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}

func (g *Google) parse(r io.Reader) ([]Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var results []Result

	// Google wraps each result in a <div class="g"> or similar.
	// We look for <a> tags inside these containers with an <h3> child.
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "g") {
			result := g.extractResult(n)
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

func (g *Google) extractResult(div *html.Node) Result {
	result := Result{Source: g.Name()}

	// Find the first <a> with an href
	var findLink func(*html.Node)
	findLink = func(n *html.Node) {
		if result.URL != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") {
					result.URL = attr.Val
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findLink(c)
		}
	}
	findLink(div)

	// Find <h3> for title
	var findH3 func(*html.Node)
	findH3 = func(n *html.Node) {
		if result.Title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "h3" {
			result.Title = textContent(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findH3(c)
		}
	}
	findH3(div)

	// Find snippet: typically in a <div> with class containing "VwiC3b" or similar.
	// As a fallback, get text from spans that aren't the title.
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if result.Snippet != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "data-sncf" || (attr.Key == "class" && strings.Contains(attr.Val, "VwiC3b")) {
					result.Snippet = textContent(n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSnippet(c)
		}
	}
	findSnippet(div)

	return result
}
