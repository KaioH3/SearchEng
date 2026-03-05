package engine

import (
	"testing"
)

func TestBrave_Name(t *testing.T) {
	b := &Brave{}
	if b.Name() != "Brave" {
		t.Errorf("Name() = %q, want 'Brave'", b.Name())
	}
}

func TestBrave_SearchWithoutAPIKey(t *testing.T) {
	b := &Brave{APIKey: ""}
	_, err := b.Search("test", 1)
	if err == nil {
		t.Error("expected error when API key is empty")
	}
}
