package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/svdx9/go-podcaster/internal/config"
	"github.com/svdx9/go-podcaster/internal/db"
	"github.com/svdx9/go-podcaster/internal/episode/api"
	"github.com/svdx9/go-podcaster/internal/episode/service"
	"github.com/svdx9/go-podcaster/internal/feed"
	"github.com/svdx9/go-podcaster/internal/http"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	slog.Info("config loaded", "config", cfg.Redacted())

	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	episodeRepo := db.NewEpisodeRepository(database)

	svc := service.New(episodeRepo, cfg.UploadDir)

	handler := api.New(svc)

	feedGen := feed.New(
		episodeRepo,
		cfg.BaseURLWithPort(),
		cfg.PodcastTitle,
		cfg.PodcastDescription,
		cfg.PodcastAuthor,
		cfg.PodcastLanguage,
		cfg.PodcastCategory,
		cfg.PodcastImageURL,
	)

	server := http.New(cfg, handler, feedGen)

	if err := server.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
