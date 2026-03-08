package engine

import (
	"net/url"
	"regexp"
	"strings"
)

var definitionPattern = regexp.MustCompile(`(?i)\b(is a|is an|are a|are an|was a|was an|refers to|defined as|significa|é um|é uma|são)\b`)

var nsfwDomains = map[string]bool{
	"xvideos.com":   true, "pornhub.com": true, "xhamster.com": true, "redtube.com": true,
	"xnxx.com":      true, "youporn.com": true, "tube8.com": true, "spankbang.com": true,
	"beeg.com":      true, "hentai.tv": true, "nhentai.net": true, "rule34.xxx": true,
	"chaturbate.com": true, "onlyfans.com": true, "brazzers.com": true,
	"livejasmin.com": true, "stripchat.com": true, "bongacams.com": true,
}

// explicitPattern matches explicit keywords with word boundaries to avoid false positives
// (e.g. "anal" should not match "analysis", "escort" should not match "escorted").
// Prefix-style entries like "masturbat" and "ejaculat" match any word starting with them.
var explicitPattern = regexp.MustCompile(`(?i)\b(porn|xxx|hentai|milf|orgasm|nude|naked|nsfw|anal|blowjob|handjob|creampie|gangbang|bukkake|masturbat\w*|ejaculat\w*|fetish|bdsm|escort|camgirl|onlyfans)\b`)

func isNSFWSource(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	if nsfwDomains[host] {
		return true
	}
	// Check subdomains (e.g. "www.pornhub.com")
	for d := range nsfwDomains {
		if strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func containsExplicit(text string) bool {
	return explicitPattern.MatchString(text)
}

// ExtractAnswer picks the best sentence from top results that directly answers the query.
// When safeSearch is true, NSFW sources and explicit content are skipped.
func ExtractAnswer(query string, results []Result, safeSearch bool) string {
	if len(results) == 0 {
		return ""
	}

	top := results
	if len(top) > 5 {
		top = top[:5]
	}

	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		return ""
	}

	var bestSentence string
	bestScore := 0.0

	for rank, r := range top {
		if safeSearch && isNSFWSource(r.URL) {
			continue
		}

		sentences := segmentSentences(r.Snippet)
		for _, sent := range sentences {
			if safeSearch && containsExplicit(sent) {
				continue
			}
			score := scoreSentence(sent, queryTerms, rank)
			if score > bestScore {
				bestScore = score
				bestSentence = sent
			}
		}
	}

	if bestScore < 0.3 {
		return ""
	}

	if runes := []rune(bestSentence); len(runes) > 300 {
		bestSentence = string(runes[:300])
		// Try to cut at last space
		if idx := strings.LastIndex(bestSentence, " "); idx > 200 {
			bestSentence = bestSentence[:idx]
		}
		bestSentence += "..."
	}

	return bestSentence
}

func segmentSentences(text string) []string {
	if text == "" {
		return nil
	}

	var sentences []string
	var current strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		if (runes[i] == '.' || runes[i] == '!' || runes[i] == '?') &&
			i+1 < len(runes) && runes[i+1] == ' ' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Remainder
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}

	return sentences
}

func scoreSentence(sentence string, queryTerms []string, resultRank int) float64 {
	words := strings.Fields(sentence)
	wordCount := len(words)

	// Length penalty: too short or too long
	if wordCount < 3 {
		return 0
	}

	// Query overlap: fraction of query terms present
	sentLower := strings.ToLower(sentence)
	matchCount := 0
	for _, term := range queryTerms {
		if strings.Contains(sentLower, term) {
			matchCount++
		}
	}
	queryOverlap := float64(matchCount) / float64(len(queryTerms))

	// Length penalty
	lengthPenalty := 1.0
	if wordCount < 5 {
		lengthPenalty = 0.5
	} else if wordCount > 50 {
		lengthPenalty = 0.6
	}

	// Result rank boost (top result gets more weight)
	rankBoost := 1.0 - float64(resultRank)*0.1

	// Definition pattern bonus
	defBonus := 0.0
	if definitionPattern.MatchString(sentence) {
		defBonus = 0.3
	}

	// Subject match: first word(s) of sentence match query terms
	subjectBonus := 0.0
	firstWords := strings.ToLower(strings.Join(words[:min(3, len(words))], " "))
	for _, term := range queryTerms {
		if strings.Contains(firstWords, term) {
			subjectBonus = 0.2
			break
		}
	}

	return queryOverlap*0.4*lengthPenalty*rankBoost + defBonus + subjectBonus
}
