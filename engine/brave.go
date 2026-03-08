package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Brave searches using the Brave Search API (free tier: 2000 queries/month).
type Brave struct {
	APIKey string
	Client *http.Client
}

func (b *Brave) Name() string { return "Brave" }

func (b *Brave) Search(ctx context.Context, query string, page int) ([]Result, error) {
	if b.APIKey == "" {
		return nil, fmt.Errorf("brave: API key not configured")
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("count", "10")
	if page > 1 {
		params.Set("offset", fmt.Sprintf("%d", (page-1)*10))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.search.brave.com/res/v1/web/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.APIKey)

	resp, err := b.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("brave: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave: status %d", resp.StatusCode)
	}

	var apiResp braveAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("brave: decode error: %w", err)
	}

	var results []Result
	for _, r := range apiResp.Web.Results {
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
			Source:  b.Name(),
		})
	}
	return results, nil
}

func (b *Brave) client() *http.Client {
	if b.Client != nil {
		return b.Client
	}
	return http.DefaultClient
}

type braveAPIResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}
