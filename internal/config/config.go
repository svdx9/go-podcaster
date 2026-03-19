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
	cfg.BaseURL = getEnv("BASE_URL", "")
	cfg.PodcastTitle = getEnv("PODCAST_TITLE", "")
	cfg.PodcastDescription = getEnv("PODCAST_DESCRIPTION", "")
	cfg.PodcastAuthor = getEnv("PODCAST_AUTHOR", "")

	// Optional fields with defaults
	port, err := getEnvInt("PORT", 8080)
	if err != nil {
		return Config{}, fmt.Errorf("config validation failed: %w", err)
	}
	cfg.Port = port

	cfg.DBPath = getEnv("DB_PATH", "./podcast.db")
	cfg.UploadDir = getEnv("UPLOAD_DIR", "./uploads")
	cfg.PodcastLanguage = getEnv("PODCAST_LANGUAGE", "en-us")
	cfg.PodcastCategory = getEnv("PODCAST_CATEGORY", "Technology")
	cfg.PodcastImageURL = getEnv("PODCAST_IMAGE_URL", "")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	// Validate required fields are not empty
	err = validateRequired(&cfg)
	if err != nil {
		return Config{}, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// getEnv retrieves an environment variable with optional requirement check.
func getEnv(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// getEnvInt retrieves an environment variable as an integer with a default value.
func getEnvInt(key string, defaultValue int) (int, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, err
	}
	return val, nil
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
