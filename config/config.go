package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration.
type Config struct {
	BraveAPIKey string
	Port        int
	Timeout     time.Duration
	MaxResults  int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	cfg := Config{
		BraveAPIKey: os.Getenv("BRAVE_API_KEY"),
		Port:        8080,
		Timeout:     5 * time.Second,
		MaxResults:  20,
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

	return cfg
}
