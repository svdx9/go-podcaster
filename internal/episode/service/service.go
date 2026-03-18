package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/audio"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
)

var (
	ErrMissingTitle     = errors.New("missing title")
	ErrUnsupportedMedia = errors.New("unsupported media type")
)

type UploadRequest struct {
	Title       string
	Description string
	Author      string
	PubDate     string
	File        io.Reader
	Filename    string
	FileSize    int64
}

type Service struct {
	repo      repository.Repository
	uploadDir string
}

func New(repo repository.Repository, uploadDir string) *Service {
	return &Service{
		repo:      repo,
		uploadDir: uploadDir,
	}
}

func (s *Service) Upload(ctx context.Context, req UploadRequest) (repository.Episode, error) {
	if req.Description == "" {
		return repository.Episode{}, errors.New("description is required")
	}
	if req.File == nil {
		return repository.Episode{}, errors.New("file is required")
	}

	data, ok := req.File.(io.ReadSeeker)
	if !ok {
		return repository.Episode{}, errors.New("file must be seekable")
	}

	header := make([]byte, 512)
	if _, err := data.Read(header); err != nil && err != io.EOF {
		return repository.Episode{}, fmt.Errorf("failed to read file header: %w", err)
	}
	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return repository.Episode{}, fmt.Errorf("failed to seek file: %w", err)
	}

	mime, err := audio.DetectMIME(header, req.Filename)
	if err != nil {
		return repository.Episode{}, ErrUnsupportedMedia
	}

	meta, err := audio.Extract(data)
	if err != nil {
		return repository.Episode{}, fmt.Errorf("failed to extract metadata: %w", err)
	}
	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return repository.Episode{}, fmt.Errorf("failed to seek file: %w", err)
	}

	title := req.Title
	if title == "" {
		title = meta.Title
	}
	if title == "" {
		return repository.Episode{}, ErrMissingTitle
	}

	episodeUUID := uuid.New().String()
	episodeDir := filepath.Join(s.uploadDir, episodeUUID)
	if err := os.MkdirAll(episodeDir, 0o755); err != nil {
		return repository.Episode{}, fmt.Errorf("failed to create episode directory: %w", err)
	}

	ext := filepath.Ext(req.Filename)
	filePath := filepath.Join(episodeDir, episodeUUID+ext)
	file, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(episodeDir)
		return repository.Episode{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, data)
	if err != nil {
		file.Close()
		os.RemoveAll(episodeDir)
		return repository.Episode{}, fmt.Errorf("failed to write file: %w", err)
	}

	episode := repository.Episode{
		UUID:         episodeUUID,
		Title:        title,
		Description:  req.Description,
		Author:       req.Author,
		FilePath:     filePath,
		FileName:     episodeUUID + ext,
		FileSize:     written,
		MimeType:     mime,
		DurationSecs: meta.DurationSecs,
	}

	if req.Author == "" && meta.Artist != "" {
		episode.Author = meta.Artist
	}

	if err := s.repo.Insert(ctx, episode); err != nil {
		file.Close()
		os.RemoveAll(episodeDir)
		return repository.Episode{}, fmt.Errorf("failed to insert episode: %w", err)
	}

	return episode, nil
}

func (s *Service) Delete(ctx context.Context, uuid string) error {
	_, err := s.repo.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, uuid); err != nil {
		return err
	}

	episodeDir := filepath.Join(s.uploadDir, uuid)
	if err := os.RemoveAll(episodeDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove episode directory: %w", err)
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
