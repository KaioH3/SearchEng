package engine

import (
	"context"
	"log/slog"
	"math"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// RankingWeights controls the scoring formula for result ranking.
type RankingWeights struct {
	PositionW          float64
	BM25W              float64
	MultiSourceW       float64
	SnippetW           float64
	TrustedDomainBonus float64
	TLDWeight          float64
	HTTPSBonus         float64
}

// Engine is the meta-search aggregator that queries multiple providers in parallel.
type Engine struct {
	Providers  []Provider
	Timeout    time.Duration
	MaxResults int
	Cache      *Cache
	Ranking    RankingWeights
	SafeSearch bool
}

// providerResult holds results from a single provider along with any error.
type providerResult struct {
	provider string
	results  []Result
	err      error
}

// SearchOptions allows per-request overrides.
type SearchOptions struct {
	SafeSearch *bool
}

// Search queries all providers in parallel, deduplicates, and ranks results.
func (e *Engine) Search(ctx context.Context, query string, page int, opts ...SearchOptions) SearchResponse {
	safeSearch := e.SafeSearch
	for _, o := range opts {
		if o.SafeSearch != nil {
			safeSearch = *o.SafeSearch
		}
	}

	// Check cache first (key includes safeSearch)
	if e.Cache != nil {
		if cached, ok := e.Cache.Get(query, page, safeSearch); ok {
			cached.Duration = 0
			return cached
		}
	}

	start := time.Now()
	timeout := e.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	searchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan providerResult, len(e.Providers))
	var wg sync.WaitGroup

	for _, p := range e.Providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			results, err := p.Search(searchCtx, query, page)
			ch <- providerResult{provider: p.Name(), results: results, err: err}
		}(p)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results until all done or context cancelled
	var allResults []providerResult
	var statuses []ProviderStatus

collecting:
	for i := 0; i < len(e.Providers); i++ {
		select {
		case pr, ok := <-ch:
			if !ok {
				break collecting
			}
			status := ProviderStatus{
				Name:    pr.provider,
				Success: pr.err == nil,
				Count:   len(pr.results),
			}
			if pr.err != nil {
				status.Error = pr.err.Error()
				slog.Warn("provider error", "provider", pr.provider, "error", pr.err)
			}
			statuses = append(statuses, status)
			allResults = append(allResults, pr)
		case <-searchCtx.Done():
			slog.Warn("search timeout reached, returning partial results")
			break collecting
		}
	}

	merged := e.mergeAndRank(allResults, query)

	maxResults := e.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}
	if len(merged) > maxResults {
		merged = merged[:maxResults]
	}

	answer := ExtractAnswer(query, merged, safeSearch)
	claims := ExtractClaims(merged, safeSearch)

	resp := SearchResponse{
		Query:     query,
		Results:   merged,
		Answer:    answer,
		Claims:    claims,
		Duration:  time.Since(start),
		Page:      page,
		Providers: statuses,
	}

	if e.Cache != nil {
		e.Cache.Set(query, page, safeSearch, resp)
	}

	return resp
}

// mergeAndRank deduplicates results by URL and scores them using RRF + BM25F.
func (e *Engine) mergeAndRank(providerResults []providerResult, query string) []Result {
	type scoredResult struct {
		result  Result
		sources []string
		ranks   map[string]int // provider -> rank position (0-based)
	}

	seen := make(map[string]*scoredResult)

	for _, pr := range providerResults {
		for i, r := range pr.results {
			normalized := normalizeURL(r.URL)
			if existing, ok := seen[normalized]; ok {
				existing.sources = append(existing.sources, r.Source)
				existing.ranks[pr.provider] = i
				if len(r.Snippet) > len(existing.result.Snippet) {
					existing.result.Snippet = r.Snippet
				}
			} else {
				seen[normalized] = &scoredResult{
					result:  r,
					sources: []string{r.Source},
					ranks:   map[string]int{pr.provider: i},
				}
			}
		}
	}

	// Calculate BM25F scores
	queryTerms := tokenize(query)
	totalDocs := float64(len(seen))

	// Count document frequency for each query term
	df := make(map[string]int)
	for _, sr := range seen {
		text := strings.ToLower(sr.result.Title + " " + sr.result.Snippet + " " + sr.result.URL)
		for _, term := range queryTerms {
			if strings.Contains(text, term) {
				df[term]++
			}
		}
	}

	w := e.Ranking
	if w.PositionW == 0 && w.BM25W == 0 && w.MultiSourceW == 0 && w.SnippetW == 0 {
		w = RankingWeights{
			PositionW:          0.4,
			BM25W:              0.3,
			MultiSourceW:       0.2,
			SnippetW:           0.1,
			TrustedDomainBonus: 1.5,
			TLDWeight:          0.15,
			HTTPSBonus:         0.3,
		}
	}

	const rrfK = 60

	results := make([]Result, 0, len(seen))
	for _, sr := range seen {
		// RRF score: sum of 1/(k + rank) for each provider
		rrfScore := 0.0
		for _, rank := range sr.ranks {
			rrfScore += 1.0 / float64(rrfK+rank)
		}

		bm25 := bm25fScore(sr.result, queryTerms, df, totalDocs)

		snippetQuality := 0.0
		if len(sr.result.Snippet) > 50 {
			snippetQuality = 1.0
		}

		// Parse URL once for all trust/TLD/HTTPS checks
		parsedURL, _ := url.Parse(sr.result.URL)
		var domainBonus, tldBonus, httpsBonusVal float64
		if parsedURL != nil {
			host := strings.ToLower(parsedURL.Host)
			if ok, _ := matchTrustedHost(host); ok {
				domainBonus = w.TrustedDomainBonus
			}
			tldBonus = tldScoreFromHost(host, w.TLDWeight)
			if parsedURL.Scheme == "https" {
				httpsBonusVal = w.HTTPSBonus
			}
		}

		score := w.PositionW*rrfScore*100 + w.BM25W*bm25 + w.MultiSourceW*float64(len(sr.sources)-1)*10.0 + w.SnippetW*snippetQuality + domainBonus + tldBonus + httpsBonusVal

		// Language mismatch penalty: CJK-heavy snippet with non-CJK query
		score += languagePenalty(sr.result.Snippet, query)

		// Query coverage penalty: too few query terms present
		score += queryCoveragePenalty(sr.result, queryTerms)

		// NaN/Inf guard
		if math.IsNaN(score) || math.IsInf(score, 0) {
			score = 0
		}

		sr.result.Score = score
		if score < 0 {
			continue
		}
		sr.result.Trust = computeTrustSignals(sr.result.URL, sr.sources)
		if len(sr.sources) > 1 {
			sr.result.Source = strings.Join(sr.sources, ", ")
		}
		results = append(results, sr.result)
	}

	// NaN-safe sort: treat NaN as less than any value
	sort.Slice(results, func(i, j int) bool {
		si, sj := results[i].Score, results[j].Score
		if math.IsNaN(si) {
			return false
		}
		if math.IsNaN(sj) {
			return true
		}
		return si > sj
	})

	return results
}

func tokenize(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	var terms []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?\"'()[]{}:;")
		if len(w) > 1 && !isStopword(w) {
			terms = append(terms, w)
		}
	}
	return terms
}

// bm25fScore computes a field-weighted BM25 score treating title, snippet, and URL as separate fields.
func bm25fScore(r Result, queryTerms []string, df map[string]int, totalDocs float64) float64 {
	if len(queryTerms) == 0 || totalDocs == 0 {
		return 0
	}

	const (
		titleBoost   = 3.0
		snippetBoost = 1.0
		urlBoost     = 2.0
		k1           = 1.2
		b            = 0.75
		avgDL        = 30.0
	)

	titleWords := tokenize(r.Title)
	snippetWords := tokenize(r.Snippet)

	// Extract URL path tokens
	urlTokens := tokenizeURL(r.URL)

	// Compute per-field TF
	titleTF := termFrequencyMap(titleWords)
	snippetTF := termFrequencyMap(snippetWords)
	urlTF := termFrequencyMap(urlTokens)

	// Combined doc length estimate
	docLen := float64(len(titleWords) + len(snippetWords) + len(urlTokens))

	score := 0.0
	for _, term := range queryTerms {
		docFreq := float64(df[term])
		if docFreq == 0 {
			continue
		}

		// Weighted TF across fields
		weightedTF := titleBoost*float64(titleTF[term]) +
			snippetBoost*float64(snippetTF[term]) +
			urlBoost*float64(urlTF[term])

		if weightedTF == 0 {
			continue
		}

		idfArg := (totalDocs - docFreq + 0.5) / (docFreq + 0.5)
		if idfArg <= 0 {
			continue // term appears in more docs than total — skip
		}
		idf := math.Log(idfArg)
		if idf < 0 {
			idf = 0 // BM25+ floor: avoid negative IDF for very common terms
		}
		tfNorm := (weightedTF * (k1 + 1)) / (weightedTF + k1*(1-b+b*docLen/avgDL))
		score += idf * tfNorm
	}

	return score
}

func termFrequencyMap(words []string) map[string]int {
	tf := make(map[string]int, len(words))
	for _, w := range words {
		tf[w]++
	}
	return tf
}

func tokenizeURL(rawURL string) []string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	// Split path and host into tokens
	parts := strings.FieldsFunc(u.Path, func(r rune) bool {
		return r == '/' || r == '-' || r == '_' || r == '.'
	})
	var tokens []string
	for _, p := range parts {
		p = strings.ToLower(p)
		if len(p) > 2 {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

var trustedDomains = map[string]bool{
	"wikipedia.org":        true,
	"github.com":           true,
	"stackoverflow.com":    true,
	"docs.python.org":      true,
	"developer.mozilla.org": true,
	"go.dev":               true,
	"docs.microsoft.com":   true,
	"learn.microsoft.com":  true,
}

func matchTrustedHost(host string) (bool, string) {
	if trustedDomains[host] {
		return true, host
	}
	for d := range trustedDomains {
		if strings.HasSuffix(host, "."+d) {
			return true, d
		}
	}
	return false, ""
}

func trustedDomainBonus(rawURL string, bonus float64) float64 {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	if ok, _ := matchTrustedHost(strings.ToLower(u.Host)); ok {
		return bonus
	}
	return 0
}

func matchesTrustedDomain(rawURL string) (bool, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false, ""
	}
	return matchTrustedHost(strings.ToLower(u.Host))
}

// TLD scoring

// Known two-part country-code suffixes
var twoPartSuffixes = map[string]bool{
	"com.br": true, "co.uk": true, "co.jp": true, "com.au": true,
	"co.in": true, "com.mx": true, "co.kr": true, "com.ar": true,
	"com.cn": true, "co.za": true, "com.tr": true, "com.tw": true,
	"co.nz": true, "com.sg": true, "com.hk": true, "com.pk": true,
	"org.uk": true, "org.br": true, "org.au": true, "ac.uk": true,
	"edu.br": true, "gov.br": true,
}

var tldScoreMap = map[string]float64{
	".edu": 1.0, ".gov": 1.0, ".mil": 1.0,
	".org": 0.5, ".ac": 0.5,
	".com": 0.0, ".net": 0.0, ".dev": 0.0, ".io": 0.0,
	".xyz": -0.5, ".click": -0.5, ".top": -0.5, ".buzz": -0.5,
	".info": -0.3, ".biz": -0.3,
}

var tldCategoryMap = map[string]string{
	".edu": "academic", ".gov": "government", ".mil": "military",
	".org": "nonprofit", ".ac": "academic",
	".com": "commercial", ".net": "commercial", ".dev": "developer", ".io": "developer",
	".xyz": "spam-prone", ".click": "spam-prone", ".top": "spam-prone", ".buzz": "spam-prone",
	".info": "low-trust", ".biz": "low-trust",
}

func extractTLD(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return extractTLDFromHost(strings.ToLower(u.Host))
}

func extractTLDFromHost(host string) string {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Strip brackets from IPv6
	host = strings.Trim(host, "[]")

	// If host is an IP address, there's no TLD
	if net.ParseIP(host) != nil {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	// Check for two-part suffixes
	if len(parts) >= 3 {
		twopart := parts[len(parts)-2] + "." + parts[len(parts)-1]
		if twoPartSuffixes[twopart] {
			return "." + twopart
		}
	}

	return "." + parts[len(parts)-1]
}

func tldCategory(tld string) string {
	if cat, ok := tldCategoryMap[tld]; ok {
		return cat
	}
	// For ccTLDs and two-part TLDs, extract the effective TLD
	parts := strings.Split(strings.TrimPrefix(tld, "."), ".")
	if len(parts) > 1 {
		effective := "." + parts[0]
		if cat, ok := tldCategoryMap[effective]; ok {
			return cat
		}
	}
	return "other"
}

func tldScore(rawURL string, weight float64) float64 {
	tld := extractTLD(rawURL)
	return tldScoreFromTLD(tld, weight)
}

func tldScoreFromHost(host string, weight float64) float64 {
	tld := extractTLDFromHost(host)
	return tldScoreFromTLD(tld, weight)
}

func tldScoreFromTLD(tld string, weight float64) float64 {
	if tld == "" {
		return 0
	}
	if s, ok := tldScoreMap[tld]; ok {
		return s * weight
	}
	parts := strings.Split(strings.TrimPrefix(tld, "."), ".")
	if len(parts) > 1 {
		base := "." + parts[0]
		if s, ok := tldScoreMap[base]; ok {
			return s * weight
		}
	}
	return 0
}

func httpsBonus(rawURL string, bonus float64) float64 {
	if strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return bonus
	}
	return 0
}

func computeTrustSignals(rawURL string, sources []string) *TrustSignals {
	tld := extractTLD(rawURL)
	isTrusted, trustedDomain := matchesTrustedDomain(rawURL)

	return &TrustSignals{
		IsHTTPS:       strings.HasPrefix(strings.ToLower(rawURL), "https://"),
		TLD:           tld,
		TLDCategory:   tldCategory(tld),
		IsTrusted:     isTrusted,
		TrustedDomain: trustedDomain,
		SourceCount:   len(sources),
	}
}

// cjkRatio returns the fraction of runes in text that are CJK characters.
func cjkRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	total := 0
	cjk := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			total++
			if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hiragana, r) {
				cjk++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(cjk) / float64(total)
}

// languagePenalty applies a penalty when snippet has >30% CJK chars but query does not.
func languagePenalty(snippet, query string) float64 {
	if cjkRatio(query) > 0.1 {
		return 0 // query itself is CJK, no penalty
	}
	if cjkRatio(snippet) > 0.3 {
		return -3.0
	}
	return 0
}

// queryCoveragePenalty penalizes results where few query terms appear.
func queryCoveragePenalty(r Result, queryTerms []string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}
	text := strings.ToLower(r.Title + " " + r.Snippet)
	matched := 0
	for _, term := range queryTerms {
		if strings.Contains(text, term) {
			matched++
		}
	}
	coverage := float64(matched) / float64(len(queryTerms))
	if coverage < 0.15 {
		return -2.0
	}
	if len(queryTerms) >= 3 && coverage < 0.34 {
		return -1.0
	}
	return 0
}

// normalizeURL strips trailing slashes, fragments, and lowercases the host for deduplication.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return strings.TrimRight(rawURL, "/")
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}
