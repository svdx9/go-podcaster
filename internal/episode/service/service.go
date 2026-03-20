package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/audio"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
	"github.com/svdx9/go-podcaster/internal/file"
)

var (
	ErrMissingTitle       = errors.New("missing title")
	ErrUnsupportedMedia   = errors.New("unsupported media type")
	ErrMissingDescription = errors.New("description is required")
	ErrMissingFile        = errors.New("file is required")
	ErrFileNotSeekable    = errors.New("file must be seekable")
)

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
	logger *slog.Logger
	repo   repository.Repository
	store  file.Store
}

func New(logger *slog.Logger, repo repository.Repository, store file.Store) *Service {
	return &Service{
		logger: logger,
		repo:   repo,
		store:  store,
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

	meta, err := audio.ReadTags(req.File)
	if err != nil {
		return repository.Episode{}, fmt.Errorf("failed to read tags: %w", err)
	}
	s.logger.Debug("meta_extraction", "meta", meta)
	_, seekErr = req.File.Seek(0, io.SeekStart)
	if seekErr != nil {
		return repository.Episode{}, fmt.Errorf("failed to seek file: %w", seekErr)
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

	// write to storage
	fileId, written, err := s.store.Save(req.File)
	if err != nil {
		return repository.Episode{}, err
	}

	episode := repository.Episode{
		UUID:         fileId,
		Title:        title,
		Description:  req.Description,
		Author:       req.Author,
		PubDate:      pubDate,
		FilePath:     "",
		FileName:     req.Filename,
		FileSize:     written,
		MimeType:     mime,
		DurationSecs: 0,
		CreatedAt:    time.Now(),
	}

	if req.Author == "" && meta.Artist != "" {
		episode.Author = meta.Artist
	}

	err = s.repo.Insert(ctx, episode)
	if err != nil {
		_ = s.store.Delete(fileId)
		return repository.Episode{}, fmt.Errorf("failed to insert episode: %w", err)
	}

	return episode, nil
}

func (s *Service) Delete(ctx context.Context, uuid uuid.UUID) error {
	ep, err := s.repo.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, uuid)
	if err != nil {
		return err
	}

	err = s.store.Delete(ep.UUID)
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
