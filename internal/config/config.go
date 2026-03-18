package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var errMissingEnvVars = errors.New("missing required environment variables")

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port               int
	DBPath             string
	UploadDir          string
	BaseURL            string
	PodcastTitle       string
	PodcastDescription string
	PodcastAuthor      string
	PodcastLanguage    string
	PodcastCategory    string
	PodcastImageURL    string
	LogLevel           string
}

// Load reads environment variables and returns a validated Config.
func Load() (Config, error) {
	var cfg Config

	// Required fields
	cfg.BaseURL = getEnv("BASE_URL", true)
	cfg.PodcastTitle = getEnv("PODCAST_TITLE", true)
	cfg.PodcastDescription = getEnv("PODCAST_DESCRIPTION", true)
	cfg.PodcastAuthor = getEnv("PODCAST_AUTHOR", true)

	// Optional fields with defaults
	cfg.Port = getEnvInt("PORT", 8080)
	cfg.DBPath = getEnv("DB_PATH", false, "./podcast.db")
	cfg.UploadDir = getEnv("UPLOAD_DIR", false, "./uploads")
	cfg.PodcastLanguage = getEnv("PODCAST_LANGUAGE", false, "en-us")
	cfg.PodcastCategory = getEnv("PODCAST_CATEGORY", false, "Technology")
	cfg.PodcastImageURL = getEnv("PODCAST_IMAGE_URL", false, "")
	cfg.LogLevel = getEnv("LOG_LEVEL", false, "info")

	// Validate required fields are not empty
	err := validateRequired(&cfg)
	if err != nil {
		return Config{}, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// getEnv retrieves an environment variable with optional requirement check.
func getEnv(key string, required bool, defaults ...string) string {
	val := os.Getenv(key)
	if val == "" && required {
		return ""
	}
	if val == "" && len(defaults) > 0 {
		return defaults[0]
	}
	return val
}

// getEnvInt retrieves an environment variable as an integer with a default value.
func getEnvInt(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// validateRequired checks that all required configuration fields are present.
func validateRequired(cfg *Config) error {
	var missing []string

	if cfg.BaseURL == "" {
		missing = append(missing, "BASE_URL")
	}
	if cfg.PodcastTitle == "" {
		missing = append(missing, "PODCAST_TITLE")
	}
	if cfg.PodcastDescription == "" {
		missing = append(missing, "PODCAST_DESCRIPTION")
	}
	if cfg.PodcastAuthor == "" {
		missing = append(missing, "PODCAST_AUTHOR")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", errMissingEnvVars, strings.Join(missing, ", "))
	}

	return nil
}

// BaseURLWithPort returns the base URL with port appended if it's not the standard port for the URL's scheme.
func (c Config) BaseURLWithPort() (string, error) {
	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil {
		// If URL parsing fails, return the original BaseURL
		return "", err
	}

	// Determine the standard port for the URL's scheme
	standardPort := 80 // default for http
	if parsedURL.Scheme == "https" {
		standardPort = 443
	}

	// Only add port if it's not the standard port for this scheme
	if c.Port != standardPort {
		parsedURL.Host = net.JoinHostPort(parsedURL.Hostname(), strconv.Itoa(c.Port))
	}

	return parsedURL.String(), nil
}

// Redacted returns a string representation of the config with sensitive information removed.
func (c Config) Redacted() string {
	return fmt.Sprintf("Config{Port=%d, DBPath=%s, UploadDir=%s, BaseURL=%s, PodcastTitle=%s, PodcastDescription=%s, PodcastAuthor=%s, PodcastLanguage=%s, PodcastCategory=%s, PodcastImageURL=%s, LogLevel=%s}",
		c.Port, c.DBPath, c.UploadDir, c.BaseURL, c.PodcastTitle, c.PodcastDescription, c.PodcastAuthor, c.PodcastLanguage, c.PodcastCategory, c.PodcastImageURL, c.LogLevel)
}
