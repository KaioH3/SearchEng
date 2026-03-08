package engine

import (
	"regexp"
	"strings"
)

var (
	numberPattern      = regexp.MustCompile(`\d+%?`)
	datePattern        = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	comparisonPattern  = regexp.MustCompile(`(?i)\b(more than|less than|faster|slower|better|worse|larger|smaller|greater|higher|lower)\b`)
	attributionPattern = regexp.MustCompile(`(?i)\b(according to|reported by|published by|stated by|announced by)\b`)
)

// ExtractClaims extracts factual claims from search results and corroborates them across sources.
// When safeSearch is true, NSFW sources and explicit content are skipped.
func ExtractClaims(results []Result, safeSearch bool) []Claim {
	type candidate struct {
		text    string
		source  string
		tokens  []string
		strength float64
	}

	var candidates []candidate

	for _, r := range results {
		if safeSearch && isNSFWSource(r.URL) {
			continue
		}

		sentences := segmentSentences(r.Snippet)
		for _, sent := range sentences {
			if safeSearch && containsExplicit(sent) {
				continue
			}
			strength := claimStrength(sent)
			if strength == 0 {
				continue
			}
			candidates = append(candidates, candidate{
				text:     sent,
				source:   r.Source,
				tokens:   tokenize(sent),
				strength: strength,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Group similar claims by Jaccard similarity
	type claimGroup struct {
		representative candidate
		sources        map[string]bool
	}

	var groups []claimGroup

	for _, c := range candidates {
		merged := false
		for i := range groups {
			if jaccardSimilarity(c.tokens, groups[i].representative.tokens) > 0.4 {
				groups[i].sources[c.source] = true
				// Keep the longer version as representative
				if len(c.text) > len(groups[i].representative.text) {
					groups[i].representative = c
				}
				merged = true
				break
			}
		}
		if !merged {
			groups = append(groups, claimGroup{
				representative: c,
				sources:        map[string]bool{c.source: true},
			})
		}
	}

	var claims []Claim
	for _, g := range groups {
		var sources []string
		for s := range g.sources {
			sources = append(sources, s)
		}

		corroboration := len(sources)
		trustedBonus := 0.0
		for _, s := range sources {
			// Check if any result from this source has a trusted URL
			// Source field may be comma-joined (e.g. "DuckDuckGo, Bing") for multi-source results
			for _, r := range results {
				if r.Source == s || strings.Contains(r.Source, s) {
					trusted, _ := matchesTrustedDomain(r.URL)
					if trusted {
						trustedBonus = 1.0
						break
					}
				}
			}
		}

		confidence := float64(corroboration)*0.3 + trustedBonus*0.2 + g.representative.strength*0.2
		if confidence > 1.0 {
			confidence = 1.0
		}

		claims = append(claims, Claim{
			Text:          g.representative.text,
			Sources:       sources,
			Corroboration: corroboration,
			Confidence:    confidence,
		})
	}

	return claims
}

// claimStrength returns how strongly a sentence looks like a factual claim (0 = not a claim).
func claimStrength(sentence string) float64 {
	words := strings.Fields(sentence)
	if len(words) < 4 {
		return 0
	}

	strength := 0.0

	if numberPattern.MatchString(sentence) {
		strength += 0.4
	}
	if datePattern.MatchString(sentence) {
		strength += 0.3
	}
	if definitionPattern.MatchString(sentence) {
		strength += 0.5
	}
	if comparisonPattern.MatchString(sentence) {
		strength += 0.3
	}
	if attributionPattern.MatchString(sentence) {
		strength += 0.4
	}

	return strength
}

// jaccardSimilarity computes the Jaccard similarity coefficient between two token sets.
func jaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	setA := make(map[string]bool, len(a))
	for _, tok := range a {
		setA[tok] = true
	}

	setB := make(map[string]bool, len(b))
	for _, tok := range b {
		setB[tok] = true
	}

	intersection := 0
	for tok := range setA {
		if setB[tok] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}
