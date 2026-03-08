package engine

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/net/html"
)

// Google searches via HTML scraping of google.com.
type Google struct {
	Client *http.Client

	mu            sync.Mutex
	cooldownUntil time.Time
}

func (g *Google) Name() string { return "Google" }

// ErrCaptcha is returned when Google serves a CAPTCHA page.
var ErrCaptcha = fmt.Errorf("google: CAPTCHA detected")

func (g *Google) Search(ctx context.Context, query string, page int) ([]Result, error) {
	// Check cooldown
	g.mu.Lock()
	if time.Now().Before(g.cooldownUntil) {
		remaining := time.Until(g.cooldownUntil).Round(time.Second)
		g.mu.Unlock()
		return nil, fmt.Errorf("google: cooling down (%s remaining)", remaining)
	}
	g.mu.Unlock()

	params := url.Values{}
	params.Set("q", query)
	params.Set("num", "10")
	if page > 1 {
		params.Set("start", fmt.Sprintf("%d", (page-1)*10))
	}

	// Detect query language for hl/gl params
	hl, gl := "en", "us"
	if hasAccentedChars(query) {
		hl, gl = "pt-BR", "br"
	}
	params.Set("hl", hl)
	params.Set("gl", gl)

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	setBrowserHeaders(req)

	// Set Accept-Language matching the query language
	if hl == "pt-BR" {
		req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	}

	// Google consent bypass
	req.Header.Set("Cookie", "CONSENT=YES+cb; SOCS=CAISNQgDEitib3FfaWRlbnRpdHlmcm9udGVuZHVpc2VydmVyXzIwMjMxMTI3LjA3X3AxGgJlbiACGgYIgJnSmgY")

	resp, err := g.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}
	defer resp.Body.Close()

	// Detect /sorry/ redirect (CAPTCHA via redirect)
	if resp.Request != nil && resp.Request.URL != nil {
		if strings.Contains(resp.Request.URL.Path, "/sorry/") {
			g.startCooldown()
			go openBrowser(resp.Request.URL.String())
			slog.Warn("google: CAPTCHA redirect detected, opening browser for manual solve")
			return nil, ErrCaptcha
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		g.startCooldown()
		return nil, fmt.Errorf("google: rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google: status %d", resp.StatusCode)
	}

	results, err := g.parse(resp.Body)
	if err != nil {
		if err == ErrCaptcha {
			g.startCooldown()
		}
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("google: 0 results (possibly blocked/CAPTCHA)")
	}

	return results, nil
}

func (g *Google) startCooldown() {
	g.mu.Lock()
	g.cooldownUntil = time.Now().Add(5 * time.Minute)
	g.mu.Unlock()
	slog.Warn("google: starting 5-minute cooldown")
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

	// Detect CAPTCHA page
	if g.detectCaptcha(doc) {
		return nil, ErrCaptcha
	}

	var results []Result

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

	// Fallback: try alternate selectors if primary found nothing
	if len(results) == 0 {
		results = g.fallbackExtract(doc)
	}

	return results, nil
}

func (g *Google) detectCaptcha(doc *html.Node) bool {
	var found bool
	var f func(*html.Node)
	f = func(n *html.Node) {
		if found {
			return
		}
		if n.Type == html.ElementNode && n.Data == "form" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "captcha-form" {
					found = true
					return
				}
				if attr.Key == "action" && strings.Contains(attr.Val, "sorry") {
					found = true
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return found
}

// fallbackExtract tries alternate CSS class patterns Google may use.
func (g *Google) fallbackExtract(doc *html.Node) []Result {
	var results []Result
	// Try data-sokoban, tF2Cxc, or other container classes
	fallbackClasses := []string{"tF2Cxc", "MjjYud", "hlcw0c"}
	for _, cls := range fallbackClasses {
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, cls) {
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
		if len(results) > 0 {
			break
		}
	}
	return results
}

func (g *Google) extractResult(div *html.Node) Result {
	result := Result{Source: g.Name()}

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

	// Try multiple snippet selectors
	snippetClasses := []string{"VwiC3b", "IsZvec", "aCOpRe", "s3v9rd"}
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if result.Snippet != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" || n.Type == html.ElementNode && n.Data == "span" {
			for _, attr := range n.Attr {
				if attr.Key == "data-sncf" {
					result.Snippet = textContent(n)
					return
				}
				if attr.Key == "class" {
					for _, cls := range snippetClasses {
						if strings.Contains(attr.Val, cls) {
							result.Snippet = textContent(n)
							return
						}
					}
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

// hasAccentedChars returns true if the text contains characters common in Portuguese
// (accented vowels, ç) suggesting a non-English query.
func hasAccentedChars(text string) bool {
	for _, r := range text {
		if r == 'ç' || r == 'Ç' {
			return true
		}
		if unicode.Is(unicode.Mn, r) { // combining marks (from NFD)
			return true
		}
		switch r {
		case 'á', 'à', 'â', 'ã', 'é', 'ê', 'í', 'ó', 'ô', 'õ', 'ú', 'ü',
			'Á', 'À', 'Â', 'Ã', 'É', 'Ê', 'Í', 'Ó', 'Ô', 'Õ', 'Ú', 'Ü':
			return true
		}
	}
	return false
}

// openBrowser opens a URL in the system's default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		slog.Warn("cannot open browser: unsupported platform", "os", runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		slog.Warn("failed to open browser", "error", err)
		return
	}
	go cmd.Wait()
}
