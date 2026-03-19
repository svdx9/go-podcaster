package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/audio"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
)

var (
	ErrMissingTitle        = errors.New("missing title")
	ErrZeroDurationEpisode = errors.New("zero duration episode")
	ErrUnsupportedMedia    = errors.New("unsupported media type")
	ErrMissingDescription  = errors.New("description is required")
	ErrMissingFile         = errors.New("file is required")
	ErrFileNotSeekable     = errors.New("file must be seekable")
)

// FileStore abstracts file persistence so the service has no direct
// dependency on the local filesystem.
type FileStore interface {
	// Save writes r to storage keyed by uuid+ext and returns the stored
	// path and the number of bytes written.
	Save(uuid, ext string, r io.Reader) (filePath string, written int64, err error)
	// Delete removes all stored files for uuid.
	Delete(uuid string) error
}

// LocalFileStore is the production FileStore backed by the local filesystem.
type LocalFileStore struct {
	uploadDir string
}

func NewLocalFileStore(uploadDir string) *LocalFileStore {
	return &LocalFileStore{uploadDir: uploadDir}
}

func (l *LocalFileStore) Save(uuid, ext string, r io.Reader) (string, int64, error) {
	episodeDir := filepath.Join(l.uploadDir, uuid)
	err := os.MkdirAll(episodeDir, 0o755)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create episode directory: %w", err)
	}

	filePath := filepath.Join(episodeDir, uuid+ext)
	f, err := os.Create(filePath)
	if err != nil {
		_ = os.RemoveAll(episodeDir)
		return "", 0, fmt.Errorf("failed to create file: %w", err)
	}

	written, err := io.Copy(f, r)
	closeErr := f.Close()
	if closeErr != nil && err == nil {
		err = fmt.Errorf("failed to close file: %w", closeErr)
	}
	if err != nil {
		_ = os.RemoveAll(episodeDir)
		return "", 0, err
	}

	return filePath, written, nil
}

func (l *LocalFileStore) Delete(uuid string) error {
	episodeDir := filepath.Join(l.uploadDir, uuid)
	err := os.RemoveAll(episodeDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove episode directory: %w", err)
	}
	return nil
}

type UploadRequest struct {
	Title       string
	Description string
	Author      string
	PubDate     string
	File        io.ReadSeeker
	Filename    string
	FileSize    int64
}

type Service struct {
	repo  repository.Repository
	store FileStore
}

func New(repo repository.Repository, store FileStore) *Service {
	return &Service{
		repo:  repo,
		store: store,
	}
}

func (s *Service) Upload(ctx context.Context, req UploadRequest) (repository.Episode, error) {
	if req.File == nil {
		return repository.Episode{}, ErrMissingFile
	}
	if req.Description == "" {
		return repository.Episode{}, ErrMissingDescription
	}

	header := make([]byte, 512)
	_, readErr := req.File.Read(header)
	if readErr != nil && readErr != io.EOF {
		return repository.Episode{}, fmt.Errorf("failed to read file header: %w", readErr)
	}
	_, seekErr := req.File.Seek(0, io.SeekStart)
	if seekErr != nil {
		return repository.Episode{}, fmt.Errorf("failed to seek file: %w", seekErr)
	}

	mime, err := audio.DetectMIME(header, req.Filename)
	if err != nil {
		return repository.Episode{}, ErrUnsupportedMedia
	}

	meta, err := audio.Extract(req.File)
	if err != nil {
		return repository.Episode{}, fmt.Errorf("failed to extract metadata: %w", err)
	}
	_, seekErr = req.File.Seek(0, io.SeekStart)
	if seekErr != nil {
		return repository.Episode{}, fmt.Errorf("failed to seek file: %w", seekErr)
	}

	if meta.DurationSecs == 0 {
		return repository.Episode{}, ErrZeroDurationEpisode
	}

	title := req.Title
	if title == "" {
		title = meta.Title
	}
	if title == "" {
		return repository.Episode{}, ErrMissingTitle
	}

	pubDate := time.Now()
	if req.PubDate != "" {
		parsed, parseErr := time.Parse(time.RFC3339, req.PubDate)
		if parseErr == nil {
			pubDate = parsed
		}
	}

	episodeUUID := uuid.New().String()
	ext := filepath.Ext(req.Filename)
	filePath, written, err := s.store.Save(episodeUUID, ext, req.File)
	if err != nil {
		return repository.Episode{}, err
	}

	episode := repository.Episode{
		UUID:         episodeUUID,
		Title:        title,
		Description:  req.Description,
		Author:       req.Author,
		PubDate:      pubDate,
		FilePath:     filePath,
		FileName:     episodeUUID + ext,
		FileSize:     written,
		MimeType:     mime,
		DurationSecs: meta.DurationSecs,
		CreatedAt:    time.Now(),
	}

	if req.Author == "" && meta.Artist != "" {
		episode.Author = meta.Artist
	}

	err = s.repo.Insert(ctx, episode)
	if err != nil {
		_ = s.store.Delete(episodeUUID)
		return repository.Episode{}, fmt.Errorf("failed to insert episode: %w", err)
	}

	return episode, nil
}

func (s *Service) Delete(ctx context.Context, uuid string) error {
	_, err := s.repo.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, uuid)
	if err != nil {
		return err
	}

	err = s.store.Delete(uuid)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]repository.Episode, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}
