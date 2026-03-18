package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/svdx9/go-podcaster/internal/config"
	"github.com/svdx9/go-podcaster/internal/feed"
)

type Server struct {
	cfg     config.Config
	handler http.Handler
	server  *http.Server
}

func New(cfg config.Config, episodeHandler http.Handler, feedGen *feed.Generator) *Server {
	s := &Server{
		cfg:     cfg,
		handler: nil,
		server:  nil,
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestLogger(&slogLogFormatter{}))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Mount("/v1", episodeHandler)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/feed.xml", http.StatusMovedPermanently)
	})

	r.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		err := feedGen.Render(r.Context(), w)
		if err != nil {
			slog.Error("failed to render feed", "error", err)
			s.writeNotInitializedError(w, r)
		}
	})

	r.Get("/files/{uuid}/{filename}", serveFileHandler)

	s.handler = r
	s.server = &http.Server{
		Addr:                         fmt.Sprintf(":%d", cfg.Port),
		Handler:                      r,
		ReadTimeout:                  30 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  120 * time.Second,
		MaxHeaderBytes:               0,
		TLSConfig:                    nil,
		TLSNextProto:                 nil,
		ConnState:                    nil,
		ErrorLog:                     nil,
		BaseContext:                  nil,
		ConnContext:                  nil,
		ReadHeaderTimeout:            0,
		DisableGeneralOptionsHandler: false,
		HTTP2:                        nil,
		Protocols:                    nil,
	}

	return s
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

func serveFileHandler(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	filename := chi.URLParam(r, "filename")

	filePath := "./uploads/" + uuid + "/" + filename

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
