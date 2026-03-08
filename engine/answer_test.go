package engine

import (
	"strings"
	"testing"
)

func TestExtractAnswer_BasicEnglish(t *testing.T) {
	results := []Result{
		{
			Title:   "Go programming language",
			Snippet: "Go is an open source programming language supported by Google. It makes it easy to build simple and efficient software.",
		},
		{
			Title:   "Go Tutorial",
			Snippet: "Learn Go programming with examples and exercises.",
		},
	}

	answer := ExtractAnswer("what is Go", results, false)
	if answer == "" {
		t.Fatal("expected an answer, got empty string")
	}
	if !strings.Contains(answer, "Go is an open source") {
		t.Errorf("expected answer about Go, got: %q", answer)
	}
}

func TestExtractAnswer_Portuguese(t *testing.T) {
	results := []Result{
		{
			Title:   "O que é Golang",
			Snippet: "Go é uma linguagem de programação criada pelo Google. Ela é usada para construir sistemas eficientes.",
		},
	}

	answer := ExtractAnswer("o que é golang", results, false)
	if answer == "" {
		t.Fatal("expected an answer for Portuguese query")
	}
}

func TestExtractAnswer_NoResults(t *testing.T) {
	answer := ExtractAnswer("test", nil, false)
	if answer != "" {
		t.Errorf("expected empty answer for no results, got: %q", answer)
	}
}

func TestExtractAnswer_EmptyQuery(t *testing.T) {
	results := []Result{{Snippet: "Some content here."}}
	answer := ExtractAnswer("", results, false)
	if answer != "" {
		t.Errorf("expected empty answer for empty query, got: %q", answer)
	}
}

func TestExtractAnswer_LengthLimit(t *testing.T) {
	long := strings.Repeat("Go programming is great and ", 20) + "this is the end."
	results := []Result{{
		Title:   "Go",
		Snippet: long,
	}}

	answer := ExtractAnswer("Go programming", results, false)
	if len(answer) > 310 { // 300 + "..."
		t.Errorf("answer too long: %d chars", len(answer))
	}
}

func TestExtractAnswer_LowQualitySkipped(t *testing.T) {
	results := []Result{
		{Title: "Random", Snippet: "Nothing relevant here at all about any topic."},
	}

	answer := ExtractAnswer("quantum physics explanation", results, false)
	if answer != "" {
		t.Errorf("expected empty answer for irrelevant results, got: %q", answer)
	}
}

func TestSegmentSentences(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Hello world. This is a test. Done!", 3},
		{"Single sentence without period", 1},
		{"First! Second? Third.", 3},
		{"", 0},
	}

	for _, tt := range tests {
		got := segmentSentences(tt.input)
		if len(got) != tt.want {
			t.Errorf("segmentSentences(%q) = %d sentences, want %d: %v", tt.input, len(got), tt.want, got)
		}
	}
}

func TestScoreSentence_DefinitionBonus(t *testing.T) {
	queryTerms := []string{"golang"}
	withDef := scoreSentence("Golang is a programming language designed for simplicity", queryTerms, 0)
	withoutDef := scoreSentence("Golang was released by Google in 2009 officially", queryTerms, 0)

	if withDef <= withoutDef {
		t.Errorf("definition sentence score (%f) should be > non-definition (%f)", withDef, withoutDef)
	}
}

func TestExtractAnswer_EmptySnippets(t *testing.T) {
	results := []Result{
		{Title: "Title", Snippet: ""},
		{Title: "Another", Snippet: ""},
	}
	answer := ExtractAnswer("test query", results, false)
	if answer != "" {
		t.Errorf("expected empty answer for empty snippets, got: %q", answer)
	}
}

func TestExtractAnswer_AllShortSnippets(t *testing.T) {
	results := []Result{
		{Title: "Title", Snippet: "Too short."},
		{Title: "Another", Snippet: "Also short."},
	}
	answer := ExtractAnswer("test query here", results, false)
	if answer != "" {
		t.Errorf("expected empty answer for all-short snippets, got: %q", answer)
	}
}

func TestExtractAnswer_TruncatesAtWord(t *testing.T) {
	// Build a long sentence that exceeds 300 runes
	long := "Go is a " + strings.Repeat("programming ", 30) + "language."
	results := []Result{{Title: "Go", Snippet: long}}

	answer := ExtractAnswer("Go programming", results, false)
	if answer == "" {
		t.Fatal("expected an answer")
	}
	if !strings.HasSuffix(answer, "...") {
		t.Errorf("expected truncated answer to end with '...', got: %q", answer)
	}
	// Should not cut in the middle of a word
	trimmed := strings.TrimSuffix(answer, "...")
	if strings.HasSuffix(trimmed, " ") {
		t.Errorf("answer should not end with trailing space before '...'")
	}
}

func TestExtractAnswer_UnicodeContent(t *testing.T) {
	// Portuguese content with accents
	snippet := "Go é uma linguagem de programação " + strings.Repeat("eficiente ", 35) + "e moderna."
	results := []Result{{Title: "Go", Snippet: snippet}}

	answer := ExtractAnswer("Go programação", results, false)
	if answer == "" {
		t.Fatal("expected an answer for unicode content")
	}
	// Verify it doesn't produce invalid UTF-8
	for i, r := range answer {
		if r == '\uFFFD' {
			t.Errorf("invalid rune at position %d in answer", i)
		}
	}
}

func TestSegmentSentences_Abbreviations(t *testing.T) {
	// Known limitation: abbreviations with periods will split incorrectly
	input := "Dr. Smith went to Washington. He arrived on time."
	sentences := segmentSentences(input)
	// Current implementation splits on ". " — so "Dr." splits here
	// This documents the known limitation
	if len(sentences) < 2 {
		t.Errorf("expected at least 2 sentences, got %d", len(sentences))
	}
}

func TestScoreSentence_RankBoost(t *testing.T) {
	queryTerms := []string{"golang", "programming"}
	sentence := "Golang is a great programming language for building systems"
	rank0 := scoreSentence(sentence, queryTerms, 0)
	rank4 := scoreSentence(sentence, queryTerms, 4)

	if rank0 <= rank4 {
		t.Errorf("rank 0 score (%f) should be > rank 4 score (%f)", rank0, rank4)
	}
}

func TestScoreSentence_ShortSentencePenalized(t *testing.T) {
	queryTerms := []string{"go"}
	short := scoreSentence("Go here", queryTerms, 0)
	if short != 0 {
		t.Errorf("expected 0 for very short sentence, got %f", short)
	}
}

func TestExtractAnswer_SafeSearch_SkipsNSFWDomain(t *testing.T) {
	results := []Result{
		{
			URL:     "https://xvideos.com/something",
			Title:   "NSFW Result",
			Snippet: "Go is a programming language designed for building systems.",
		},
		{
			URL:     "https://golang.org/doc",
			Title:   "Go Documentation",
			Snippet: "Go is an open source programming language supported by Google.",
		},
	}

	answer := ExtractAnswer("what is Go", results, true)
	if answer == "" {
		t.Fatal("expected an answer from safe source")
	}
	if strings.Contains(answer, "NSFW") {
		t.Errorf("expected answer from safe source, got NSFW content")
	}
}

func TestExtractAnswer_SafeSearch_SkipsExplicitContent(t *testing.T) {
	results := []Result{
		{
			URL:     "https://example.com/page",
			Title:   "Example",
			Snippet: "Go is used for porn sites and nsfw content generation.",
		},
		{
			URL:     "https://golang.org/doc",
			Title:   "Go Documentation",
			Snippet: "Go is an open source programming language supported by Google.",
		},
	}

	answer := ExtractAnswer("what is Go", results, true)
	if strings.Contains(strings.ToLower(answer), "porn") {
		t.Errorf("safe search should filter explicit content, got: %q", answer)
	}
}

func TestExtractAnswer_NoSafeSearch_AllowsAll(t *testing.T) {
	results := []Result{
		{
			URL:     "https://xvideos.com/something",
			Title:   "Result",
			Snippet: "Go is a programming language designed for building systems.",
		},
	}

	answer := ExtractAnswer("what is Go", results, false)
	// Without safe search, NSFW sources are not filtered
	if answer == "" {
		t.Fatal("expected answer when safe search is off")
	}
}

func TestIsNSFWSource(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://xvideos.com/video123", true},
		{"https://www.pornhub.com/view", true},
		{"https://golang.org/doc", false},
		{"https://github.com/user/repo", false},
	}
	for _, tt := range tests {
		got := isNSFWSource(tt.url)
		if got != tt.want {
			t.Errorf("isNSFWSource(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestContainsExplicit(t *testing.T) {
	if !containsExplicit("This contains porn content") {
		t.Error("expected true for explicit keyword")
	}
	if containsExplicit("This is a normal programming tutorial") {
		t.Error("expected false for clean content")
	}
}

func TestContainsExplicit_NoFalsePositives(t *testing.T) {
	// These should NOT be flagged — they contain substrings of explicit words
	// but are legitimate terms
	safePhrases := []string{
		"This is an analysis of the data",
		"The canal was built in 1914",
		"She escorted the diplomat to the meeting",
		"A denuded landscape after the wildfire",
		"Analog signals are used in electronics",
		"The penalty was harsh",
		"Fetching data from the API",
	}
	for _, phrase := range safePhrases {
		if containsExplicit(phrase) {
			t.Errorf("false positive: containsExplicit(%q) = true", phrase)
		}
	}
}

func TestContainsExplicit_TruePositives(t *testing.T) {
	explicitPhrases := []string{
		"free porn videos online",
		"XXX rated content",
		"hentai manga collection",
		"visit this escort service",
		"masturbation tips",
		"ejaculation problems",
		"BDSM community forum",
		"NSFW warning applies here",
	}
	for _, phrase := range explicitPhrases {
		if !containsExplicit(phrase) {
			t.Errorf("missed explicit content: containsExplicit(%q) = false", phrase)
		}
	}
}
