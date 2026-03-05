package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere
	os.Unsetenv("BRAVE_API_KEY")
	os.Unsetenv("SEARCHENG_PORT")
	os.Unsetenv("SEARCHENG_TIMEOUT")
	os.Unsetenv("SEARCHENG_MAX_RESULTS")

	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", cfg.Timeout)
	}
	if cfg.MaxResults != 20 {
		t.Errorf("MaxResults = %d, want 20", cfg.MaxResults)
	}
	if cfg.BraveAPIKey != "" {
		t.Errorf("BraveAPIKey = %q, want empty", cfg.BraveAPIKey)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("BRAVE_API_KEY", "test-key-123")
	os.Setenv("SEARCHENG_PORT", "3000")
	os.Setenv("SEARCHENG_TIMEOUT", "10s")
	os.Setenv("SEARCHENG_MAX_RESULTS", "50")
	defer func() {
		os.Unsetenv("BRAVE_API_KEY")
		os.Unsetenv("SEARCHENG_PORT")
		os.Unsetenv("SEARCHENG_TIMEOUT")
		os.Unsetenv("SEARCHENG_MAX_RESULTS")
	}()

	cfg := Load()

	if cfg.BraveAPIKey != "test-key-123" {
		t.Errorf("BraveAPIKey = %q, want 'test-key-123'", cfg.BraveAPIKey)
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Port)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", cfg.Timeout)
	}
	if cfg.MaxResults != 50 {
		t.Errorf("MaxResults = %d, want 50", cfg.MaxResults)
	}
}

func TestLoad_InvalidEnvFallsBackToDefaults(t *testing.T) {
	os.Setenv("SEARCHENG_PORT", "not-a-number")
	os.Setenv("SEARCHENG_TIMEOUT", "invalid")
	os.Setenv("SEARCHENG_MAX_RESULTS", "xyz")
	defer func() {
		os.Unsetenv("SEARCHENG_PORT")
		os.Unsetenv("SEARCHENG_TIMEOUT")
		os.Unsetenv("SEARCHENG_MAX_RESULTS")
	}()

	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080 (fallback)", cfg.Port)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s (fallback)", cfg.Timeout)
	}
	if cfg.MaxResults != 20 {
		t.Errorf("MaxResults = %d, want 20 (fallback)", cfg.MaxResults)
	}
}
