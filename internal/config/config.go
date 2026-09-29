// Package config loads and validates LogPilot runtime configuration.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains all process-level settings.
type Config struct {
	HTTPAddr        string
	DatabasePath    string
	LLMBaseURL      string
	LLMAPIKey       string
	LLMModel        string
	EmbedDimension  int
	QdrantURL       string
	ServiceTargets  map[string]string
	ShutdownTimeout time.Duration
}

// Load reads configuration from LOGPILOT_* environment variables and validates it.
func Load() (Config, error) {
	dimension, err := positiveInt("LOGPILOT_EMBED_DIMENSION")
	if err != nil {
		return Config{}, err
	}

	targets, err := parseTargets(os.Getenv("LOGPILOT_SERVICE_TARGETS"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:        valueOrDefault("LOGPILOT_HTTP_ADDR", ":8080"),
		DatabasePath:    valueOrDefault("LOGPILOT_DATABASE_PATH", "./data/logpilot.db"),
		LLMBaseURL:      strings.TrimSpace(os.Getenv("LOGPILOT_LLM_BASE_URL")),
		LLMAPIKey:       os.Getenv("LOGPILOT_LLM_API_KEY"),
		LLMModel:        strings.TrimSpace(os.Getenv("LOGPILOT_LLM_MODEL")),
		EmbedDimension:  dimension,
		QdrantURL:       valueOrDefault("LOGPILOT_QDRANT_URL", "http://qdrant:6333"),
		ServiceTargets:  targets,
		ShutdownTimeout: 15 * time.Second,
	}

	required := []struct {
		name  string
		value string
	}{
		{"LOGPILOT_LLM_BASE_URL", cfg.LLMBaseURL},
		{"LOGPILOT_LLM_MODEL", cfg.LLMModel},
	}
	for _, item := range required {
		if item.value == "" {
			return Config{}, fmt.Errorf("%s is required", item.name)
		}
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func positiveInt(key string) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = "384"
	}
	dimension, err := strconv.Atoi(value)
	if err != nil || dimension <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return dimension, nil
}

func parseTargets(raw string) (map[string]string, error) {
	targets := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return targets, nil
	}
	for _, entry := range strings.Split(raw, ",") {
		name, target, ok := strings.Cut(strings.TrimSpace(entry), "=")
		parsed, err := url.ParseRequestURI(target)
		if !ok || strings.TrimSpace(name) == "" || err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("invalid LOGPILOT_SERVICE_TARGETS entry")
		}
		targets[strings.TrimSpace(name)] = target
	}
	return targets, nil
}
