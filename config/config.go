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
	CacheTTL       time.Duration
	Ranking        RankingWeights
	GoogleRPM      float64
	DDGRPM         float64
	BingRPM        float64
	SafeSearch     bool
}

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

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	cfg := Config{
		BraveAPIKey:    os.Getenv("BRAVE_API_KEY"),
		Port:           8080,
		Timeout:        5 * time.Second,
		MaxResults:     20,
		MaxRetries:     2,
		RetryBaseDelay: 500 * time.Millisecond,
		CacheTTL:       1 * time.Hour,
		GoogleRPM:      1,
		DDGRPM:         10,
		BingRPM:        10,
		SafeSearch:     true,
		Ranking: RankingWeights{
			PositionW:          0.4,
			BM25W:              0.3,
			MultiSourceW:       0.2,
			SnippetW:           0.1,
			TrustedDomainBonus: 1.5,
			TLDWeight:          0.15,
			HTTPSBonus:         0.3,
		},
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

	if c := os.Getenv("SEARCHENG_CACHE_TTL"); c != "" {
		if ttl, err := time.ParseDuration(c); err == nil {
			cfg.CacheTTL = ttl
		}
	}

	if v := os.Getenv("SEARCHENG_SAFE_SEARCH"); v != "" {
		cfg.SafeSearch = v != "false" && v != "0"
	}

	loadFloat(&cfg.GoogleRPM, "SEARCHENG_GOOGLE_RPM")
	loadFloat(&cfg.DDGRPM, "SEARCHENG_DDG_RPM")
	loadFloat(&cfg.BingRPM, "SEARCHENG_BING_RPM")

	loadFloat(&cfg.Ranking.PositionW, "SEARCHENG_RANK_POSITION_W")
	loadFloat(&cfg.Ranking.BM25W, "SEARCHENG_RANK_BM25_W")
	loadFloat(&cfg.Ranking.MultiSourceW, "SEARCHENG_RANK_MULTISOURCE_W")
	loadFloat(&cfg.Ranking.SnippetW, "SEARCHENG_RANK_SNIPPET_W")
	loadFloat(&cfg.Ranking.TrustedDomainBonus, "SEARCHENG_RANK_TRUSTED_DOMAIN_BONUS")
	loadFloat(&cfg.Ranking.TLDWeight, "SEARCHENG_RANK_TLD_W")
	loadFloat(&cfg.Ranking.HTTPSBonus, "SEARCHENG_RANK_HTTPS_BONUS")

	return cfg
}

func loadFloat(dst *float64, envKey string) {
	if v := os.Getenv(envKey); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			*dst = f
		}
	}
}
