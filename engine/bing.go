package engine

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Bing searches via HTML scraping of bing.com.
type Bing struct {
	Client *http.Client
}

func (b *Bing) Name() string { return "Bing" }

func (b *Bing) Search(query string, page int) ([]Result, error) {
	params := url.Values{}
	params.Set("q", query)
	if page > 1 {
		params.Set("first", fmt.Sprintf("%d", (page-1)*10+1))
	}

	req, err := http.NewRequest("GET", "https://www.bing.com/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	setBrowserHeaders(req)

	resp, err := b.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("bing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing: status %d", resp.StatusCode)
	}

	return b.parse(resp.Body)
}

func (b *Bing) client() *http.Client {
	if b.Client != nil {
		return b.Client
	}
	return http.DefaultClient
}

func (b *Bing) parse(r io.Reader) ([]Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var results []Result

	// Bing results are in <li class="b_algo">
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" && hasClass(n, "b_algo") {
			result := b.extractResult(n)
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

func (b *Bing) extractResult(li *html.Node) Result {
	result := Result{Source: b.Name()}

	// Find <h2> > <a> for title and URL
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if result.Title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "h2" {
			// Find <a> inside h2
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "a" {
					result.Title = textContent(c)
					for _, attr := range c.Attr {
						if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") {
							result.URL = attr.Val
						}
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(li)

	// Find snippet in <div class="b_caption"> > <p>
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if result.Snippet != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "b_caption") {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "p" {
					result.Snippet = textContent(c)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSnippet(c)
		}
	}
	findSnippet(li)

	return result
}
