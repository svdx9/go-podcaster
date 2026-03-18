package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	openapi_types "github.com/oapi-codegen/runtime/types"
	apiv1 "github.com/svdx9/go-podcaster/internal/api/v1"
	"github.com/svdx9/go-podcaster/internal/config"
	episodeHandler "github.com/svdx9/go-podcaster/internal/episode/api"
	"github.com/svdx9/go-podcaster/internal/feed"
)

type feedGenerator interface {
	Render(ctx context.Context, w io.Writer) error
}
type Server struct {
	*episodeHandler.Handler
	cfg     config.Config
	server  *http.Server
	feedgen feedGenerator
}

func New(cfg config.Config, episodeHandler *episodeHandler.Handler, feedGen *feed.Generator) *Server {
	s := &Server{
		Handler: episodeHandler,
		cfg:     cfg,
		server:  nil,
		feedgen: feedGen,
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestLogger(&slogLogFormatter{}))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Mount("/", apiv1.HandlerFromMux(s, chi.NewMux()))

	s.server = &http.Server{
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
		s.writeNotInitializedError(w, r)
	}
}

func (s *Server) GetFilesUuidFilename(w http.ResponseWriter, r *http.Request, uuid openapi_types.UUID, filename string) {
	// Serve audio file
	// (GET /files/{uuid}/{filename})
	filePath := filepath.Join(s.cfg.UploadDir, uuid.String(), filename)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	contentType := getContentType(filename)
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, filename, stat.ModTime(), file)
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

// writeNotInitializedError writes a JSON error response when system is not initialized
func (s *Server) writeNotInitializedError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	errorResponse := `{"error": "not initialised", "message": "The podcast system has not been initialized. Please set up the database and try again."}`
	_, err := w.Write([]byte(errorResponse))
	if err != nil {
		slog.Error("failed to write error response", "error", err)
	}
}

func getContentType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".mp3"):
		return "audio/mpeg"
	case strings.HasSuffix(lower, ".m4a"):
		return "audio/x-m4a"
	case strings.HasSuffix(lower, ".mp4"):
		return "audio/mp4"
	case strings.HasSuffix(lower, ".aac"):
		return "audio/aac"
	case strings.HasSuffix(lower, ".aiff"), strings.HasSuffix(lower, ".aif"):
		return "audio/aiff"
	case strings.HasSuffix(lower, ".wav"):
		return "audio/x-wav"
	default:
		return "application/octet-stream"
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
