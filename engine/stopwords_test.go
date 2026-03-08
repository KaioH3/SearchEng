package engine

import "testing"

func TestIsStopword_English(t *testing.T) {
	stopwords := []string{"the", "is", "how", "and", "for", "with", "are", "was"}
	for _, w := range stopwords {
		if !isStopword(w) {
			t.Errorf("isStopword(%q) = false, want true", w)
		}
	}
}

func TestIsStopword_Portuguese(t *testing.T) {
	stopwords := []string{"como", "que", "para", "de", "da", "dos", "em", "uma"}
	for _, w := range stopwords {
		if !isStopword(w) {
			t.Errorf("isStopword(%q) = false, want true", w)
		}
	}
}

func TestIsStopword_ContentWord(t *testing.T) {
	content := []string{"engineer", "golang", "python", "search", "tutorial", "database"}
	for _, w := range content {
		if isStopword(w) {
			t.Errorf("isStopword(%q) = true, want false", w)
		}
	}
}
