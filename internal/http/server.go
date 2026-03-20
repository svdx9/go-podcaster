package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	apiv1 "github.com/svdx9/go-podcaster/internal/api/v1"
	"github.com/svdx9/go-podcaster/internal/config"
	"github.com/svdx9/go-podcaster/internal/db/queries"
	episodeHandler "github.com/svdx9/go-podcaster/internal/episode/api"
	"github.com/svdx9/go-podcaster/internal/feed"
	"github.com/svdx9/go-podcaster/internal/file"
)

type feedGenerator interface {
	Render(ctx context.Context, w io.Writer) error
}
type Server struct {
	*episodeHandler.Handler
	cfg       config.Config
	server    *http.Server
	feedgen   feedGenerator
	fileStore file.Store
	querier   queries.Querier
}

func New(cfg config.Config, episodeHandler *episodeHandler.Handler, feedGen *feed.Generator, querier queries.Querier, fileStore file.Store) *Server {
	s := &Server{ //nolint:exhaustruct
		Handler:   episodeHandler,
		cfg:       cfg,
		server:    nil,
		feedgen:   feedGen,
		fileStore: fileStore,
		querier:   querier,
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestLogger(&slogLogFormatter{}))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Mount("/", apiv1.HandlerFromMux(s, chi.NewMux()))

	s.server = &http.Server{ //nolint:exhaustruct
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
	}

	return s
}

func (s *Server) GetRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/feed.xml", http.StatusMovedPermanently)
}

func (s *Server) GetFeedXml(w http.ResponseWriter, r *http.Request) {
	// Get iTunes-compatible RSS feed
	// (GET /feed.xml)
	w.Header().Set("Content-Type", "application/xml")
	err := s.feedgen.Render(r.Context(), w)
	if err != nil {
		slog.Error("failed to render feed", "error", err)
		s.writeNoFeedError(w, r)
	}
}

func (s *Server) GetFilesByUuid(w http.ResponseWriter, r *http.Request, UUID uuid.UUID) {
	// Serve audio file
	// (GET /files/{uuid})

	// get content type and modtime from database

	episode, err := s.querier.GetEpisodeByUUID(r.Context(), UUID)
	if err != nil {
		// TODO: improve this
		http.Error(w, "No such episode", http.StatusInternalServerError)
		return
	}
	cb := func(file io.ReadSeeker) error {
		w.Header().Set("Content-Type", episode.MimeType)
		http.ServeContent(w, r, episode.Uuid.String(), episode.CreatedAt, file)
		return nil
	}

	err = s.fileStore.ReadSeekFile(UUID, cb)
	if err != nil {
		slog.Error("failed to read file", "error", err)
	}

}

type slogLogFormatter struct{}

func (f *slogLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &slogLogEntry{r: r}
}

type slogLogEntry struct {
	r *http.Request
}

func (e *slogLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	slog.Info("request",
		"method", e.r.Method,
		"path", e.r.URL.Path,
		"status", status,
		"bytes", bytes,
		"elapsed", elapsed.String(),
	)
}

func (e *slogLogEntry) Panic(v interface{}, stack []byte) {
	slog.Error("panic", "value", v, "stack", string(stack))
}

// writeNoFeedError writes a JSON error response when system is not initialized
func (s *Server) writeNoFeedError(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	errorResponse := `{"error": "no feed", "message": "The podcast system has no feed."}`
	_, err := w.Write([]byte(errorResponse))
	if err != nil {
		slog.Error("failed to write error response", "error", err)
	}
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		slog.Info("starting server", "addr", s.server.Addr)
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
