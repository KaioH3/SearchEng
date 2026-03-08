package engine

import (
	"strings"
	"testing"
	"time"
)

func TestGoogle_Name(t *testing.T) {
	g := &Google{}
	if g.Name() != "Google" {
		t.Errorf("Name() = %q, want 'Google'", g.Name())
	}
}

func TestGoogle_ParseHTML(t *testing.T) {
	html := `<html><body>
		<div class="g">
			<a href="https://example.com/result1"><h3>Result One</h3></a>
			<div class="VwiC3b">This is the snippet for result one</div>
		</div>
		<div class="g">
			<a href="https://example.com/result2"><h3>Result Two</h3></a>
			<div class="VwiC3b">Snippet for result two</div>
		</div>
	</body></html>`

	g := &Google{}
	results, err := g.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].URL != "https://example.com/result1" {
		t.Errorf("result[0].URL = %q, want 'https://example.com/result1'", results[0].URL)
	}
	if !strings.Contains(results[0].Title, "Result One") {
		t.Errorf("result[0].Title = %q, expected to contain 'Result One'", results[0].Title)
	}
	if results[0].Source != "Google" {
		t.Errorf("result[0].Source = %q, want 'Google'", results[0].Source)
	}
}

func TestGoogle_ParseEmptyHTML(t *testing.T) {
	g := &Google{}
	results, err := g.parse(strings.NewReader("<html><body></body></html>"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty HTML, got %d", len(results))
	}
}

func TestGoogle_ParseSkipsNoHref(t *testing.T) {
	html := `<html><body>
		<div class="g">
			<h3>Title Without Link</h3>
		</div>
	</body></html>`

	g := &Google{}
	results, err := g.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (no href), got %d", len(results))
	}
}

func TestGoogle_DetectCaptcha_SorryForm(t *testing.T) {
	html := `<html><body>
		<form action="/sorry/index">
			<input type="submit" value="Submit">
		</form>
	</body></html>`

	g := &Google{}
	results, err := g.parse(strings.NewReader(html))
	if err == nil {
		t.Fatal("expected CAPTCHA error, got nil")
	}
	if !strings.Contains(err.Error(), "CAPTCHA") {
		t.Errorf("error = %q, expected CAPTCHA message", err.Error())
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on CAPTCHA, got %d", len(results))
	}
}

func TestGoogle_DetectCaptcha_CaptchaForm(t *testing.T) {
	html := `<html><body>
		<form id="captcha-form" action="/captcha">
			<input type="text">
		</form>
	</body></html>`

	g := &Google{}
	_, err := g.parse(strings.NewReader(html))
	if err == nil {
		t.Fatal("expected CAPTCHA error")
	}
}

func TestGoogle_DetectsSorryRedirect(t *testing.T) {
	// The /sorry/ redirect is detected at the HTTP level in Search(),
	// but we can test that detectCaptcha catches the form action.
	html := `<html><body>
		<form action="/sorry/index?continue=...">
			<input type="submit">
		</form>
	</body></html>`

	g := &Google{}
	_, err := g.parse(strings.NewReader(html))
	if err == nil {
		t.Fatal("expected CAPTCHA error for /sorry/ form")
	}
	if err != ErrCaptcha {
		t.Errorf("error = %v, want ErrCaptcha", err)
	}
}

func TestGoogle_CooldownAfterCaptcha(t *testing.T) {
	g := &Google{}
	g.startCooldown()

	g.mu.Lock()
	cooldown := g.cooldownUntil
	g.mu.Unlock()

	if cooldown.Before(time.Now().Add(4 * time.Minute)) {
		t.Error("cooldown should be ~5 minutes from now")
	}
}

func TestHasAccentedChars(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", false},
		{"vaporizador de ervas", false},
		{"como começar", true},
		{"café com leite", true},
		{"programação avançada", true},
	}
	for _, tt := range tests {
		if got := hasAccentedChars(tt.input); got != tt.want {
			t.Errorf("hasAccentedChars(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGoogle_FallbackExtract(t *testing.T) {
	html := `<html><body>
		<div class="tF2Cxc">
			<a href="https://example.com/fallback"><h3>Fallback Result</h3></a>
			<div class="VwiC3b">Fallback snippet</div>
		</div>
	</body></html>`

	g := &Google{}
	results, err := g.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 fallback result, got %d", len(results))
	}
	if results[0].URL != "https://example.com/fallback" {
		t.Errorf("URL = %q, want fallback URL", results[0].URL)
	}
}
