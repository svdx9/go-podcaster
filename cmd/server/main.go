package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/svdx9/go-podcaster/internal/config"
	"github.com/svdx9/go-podcaster/internal/db"
	"github.com/svdx9/go-podcaster/internal/episode/api"
	"github.com/svdx9/go-podcaster/internal/episode/service"
	"github.com/svdx9/go-podcaster/internal/feed"
	"github.com/svdx9/go-podcaster/internal/file"
	"github.com/svdx9/go-podcaster/internal/http"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(
		slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{Level: cfg.LogLevel}, //nolint:exhaustruct
		),
	)
	slog.SetDefault(logger)

	logger.Info("config loaded", "config", cfg.Redacted())

	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		closeErr := database.Close()
		if closeErr != nil {
			logger.Error("failed to close database", "error", closeErr)
		}
	}()

	querier := db.NewQuerier(database)
	episodeRepo := db.NewEpisodeRepository(querier)

	fileStore := file.NewStore(cfg.UploadDir)
	err = fileStore.Init()
	if err != nil {
		log.Fatalf("failed to initialize file store: %v", err)
	}

	svc := service.New(logger, episodeRepo, fileStore)
	handler := api.New(logger, svc)

	baseUrl, err := cfg.BaseURLWithPort()
	if err != nil {
		log.Fatalf("failed to get base URL: %v", err)
	}

	feedGen := feed.New(
		episodeRepo,
		baseUrl,
		cfg.PodcastTitle,
		cfg.PodcastDescription,
		cfg.PodcastAuthor,
		cfg.PodcastLanguage,
		cfg.PodcastCategory,
		cfg.PodcastImageURL,
	)

	server := http.New(cfg, handler, feedGen, querier, fileStore)

	err = server.Start(ctx)
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}
