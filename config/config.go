package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration.
type Config struct {
	BraveAPIKey    string
	Port           int
	Timeout        time.Duration
	MaxResults     int
	MaxRetries     int
	RetryBaseDelay time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	cfg := Config{
		BraveAPIKey:    os.Getenv("BRAVE_API_KEY"),
		Port:           8080,
		Timeout:        5 * time.Second,
		MaxResults:     20,
		MaxRetries:     2,
		RetryBaseDelay: 500 * time.Millisecond,
	}

	if p := os.Getenv("SEARCHENG_PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			cfg.Port = port
		}
	}

	if t := os.Getenv("SEARCHENG_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			cfg.Timeout = d
		}
	}

	if m := os.Getenv("SEARCHENG_MAX_RESULTS"); m != "" {
		if max, err := strconv.Atoi(m); err == nil {
			cfg.MaxResults = max
		}
	}

	if r := os.Getenv("SEARCHENG_MAX_RETRIES"); r != "" {
		if retries, err := strconv.Atoi(r); err == nil {
			cfg.MaxRetries = retries
		}
	}

	if d := os.Getenv("SEARCHENG_RETRY_DELAY"); d != "" {
		if delay, err := time.ParseDuration(d); err == nil {
			cfg.RetryBaseDelay = delay
		}
	}

	return cfg
}
