/*
Package config handles application configuration from environment variables.

Environment Variables:

	Required:
	  HOST                  Hostname for composing absolute URLs (e.g., localhost, api.example.com)
	  BASE_URL             Base URL for the podcast feed
	  PODCAST_TITLE        Title of the podcast
	  PODCAST_DESCRIPTION  Description of the podcast
	  PODCAST_AUTHOR       Author of the podcast

	Optional:
	  PORT                 Listen port; default 8080
	  ENV                  Deployment environment: development, staging, production; default development
	  LOG_LEVEL            Logging level: DEBUG, INFO, WARN, ERROR; default INFO
	  DB_PATH              Path to SQLite database; default ./podcast.db
	  UPLOAD_DIR           Directory for uploaded files; default ./uploads
	  PODCAST_LANGUAGE     Podcast language; default en-us
	  PODCAST_CATEGORY     Podcast category; default Technology
	  PODCAST_IMAGE_URL    URL to podcast cover image

Validation:

	Config is validated at startup in FromEnv(). Invalid values cause
	immediate startup failure with a clear error message.

	Constraints:
	  - PORT must be 1-65535 if set
	  - ENV must be one of: development, staging, production
	  - LOG_LEVEL must be one of: DEBUG, INFO, WARN, ERROR
	  - BASE_URL must be a valid URL

Secrets:

	Config must NOT be logged verbatim. Use Config.Redacted() to get a
	safe representation containing only non-sensitive fields.

Immutability:

	Config is populated once at startup and must not be mutated or
	reloaded. Pass it via dependency injection to handlers.
*/
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment        string
	Port               int
	Host               string
	LogLevel           slog.Level
	DBPath             string
	UploadDir          string
	BaseURL            string
	PodcastTitle       string
	PodcastDescription string
	PodcastAuthor      string
	PodcastLanguage    string
	PodcastCategory    string
	PodcastImageURL    string
}

var (
	ErrMissingEnv      = errors.New("missing required env var")
	ErrInvalidBool     = errors.New("invalid boolean value")
	ErrInvalidInt      = errors.New("invalid integer value")
	ErrInvalidURL      = errors.New("invalid URL")
	ErrInvalidLogLevel = errors.New("invalid log level")
	ErrInvalidEnv      = errors.New("invalid environment")
	ErrPortOutOfRange  = errors.New("port out of range")
)

func fromEnvKey(key string) (string, bool) {
	return os.LookupEnv(key)
}

func getEnvOrDefault(key, defaultVal string) string {
	val, exists := fromEnvKey(key)
	if !exists || strings.TrimSpace(val) == "" {
		return defaultVal
	}
	return strings.TrimSpace(val)
}

func requireEnv(key string) (string, error) {
	val, exists := fromEnvKey(key)
	if !exists || strings.TrimSpace(val) == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingEnv, key)
	}
	return strings.TrimSpace(val), nil
}

func parsePort(val string, defaultVal int) (int, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal, nil
	}
	port, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidInt, val)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("%w: %d", ErrPortOutOfRange, port)
	}
	return port, nil
}

func parseLogLevel(val string, defaultVal slog.Level) (slog.Level, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal, nil
	}
	switch strings.ToUpper(val) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("%w: %q", ErrInvalidLogLevel, val)
	}
}

func FromEnv() (Config, error) {
	var cfg Config

	environment := getEnvOrDefault("ENV", "development")
	if environment != "development" && environment != "staging" && environment != "production" {
		return Config{}, fmt.Errorf("%w: ENV must be 'development', 'staging', or 'production', got %q", ErrInvalidEnv, environment)
	}
	cfg.Environment = environment

	var defaultLogLevel slog.Level
	switch environment {
	case "production", "staging":
		defaultLogLevel = slog.LevelInfo
	case "development":
		defaultLogLevel = slog.LevelDebug
	}

	host, err := requireEnv("HOST")
	if err != nil {
		return cfg, err
	}
	cfg.Host = host

	port, err := parsePort(getEnvOrDefault("PORT", "8080"), 8080)
	if err != nil {
		return cfg, fmt.Errorf("PORT: %w", err)
	}
	cfg.Port = port

	logLevel, err := parseLogLevel(getEnvOrDefault("LOG_LEVEL", ""), defaultLogLevel)
	if err != nil {
		return cfg, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	cfg.LogLevel = logLevel

	cfg.DBPath = getEnvOrDefault("DB_PATH", "./podcast.db")
	cfg.UploadDir = getEnvOrDefault("UPLOAD_DIR", "./uploads")

	baseURL, err := requireEnv("BASE_URL")
	if err != nil {
		return cfg, err
	}
	_, err = url.Parse(baseURL)
	if err != nil {
		return cfg, fmt.Errorf("BASE_URL: %w", err)
	}
	cfg.BaseURL = baseURL

	cfg.PodcastTitle, err = requireEnv("PODCAST_TITLE")
	if err != nil {
		return cfg, err
	}

	cfg.PodcastDescription, err = requireEnv("PODCAST_DESCRIPTION")
	if err != nil {
		return cfg, err
	}

	cfg.PodcastAuthor, err = requireEnv("PODCAST_AUTHOR")
	if err != nil {
		return cfg, err
	}

	cfg.PodcastLanguage = getEnvOrDefault("PODCAST_LANGUAGE", "en-us")
	cfg.PodcastCategory = getEnvOrDefault("PODCAST_CATEGORY", "Technology")
	cfg.PodcastImageURL = getEnvOrDefault("PODCAST_IMAGE_URL", "")

	return cfg, nil
}

type ConfigRedacted struct {
	Environment        string
	Port               int
	Host               string
	LogLevel           string
	DBPath             string
	UploadDir          string
	PodcastTitle       string
	PodcastDescription string
	PodcastAuthor      string
	PodcastLanguage    string
	PodcastCategory    string
	PodcastImageURL    string
}

func (c Config) Redacted() ConfigRedacted {
	return ConfigRedacted{
		Environment:        c.Environment,
		Port:               c.Port,
		Host:               c.Host,
		LogLevel:           c.LogLevel.String(),
		DBPath:             c.DBPath,
		UploadDir:          c.UploadDir,
		PodcastTitle:       c.PodcastTitle,
		PodcastDescription: c.PodcastDescription,
		PodcastAuthor:      c.PodcastAuthor,
		PodcastLanguage:    c.PodcastLanguage,
		PodcastCategory:    c.PodcastCategory,
		PodcastImageURL:    c.PodcastImageURL,
	}
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c Config) BaseURLWithPort() (string, error) {
	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}

	standardPort := 80
	if parsedURL.Scheme == "https" {
		standardPort = 443
	}

	if c.Port != standardPort {
		parsedURL.Host = net.JoinHostPort(parsedURL.Hostname(), strconv.Itoa(c.Port))
	}

	return parsedURL.String(), nil
}

func (c Config) ServerAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}
