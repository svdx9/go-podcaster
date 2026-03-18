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

var ErrNotFoundRepo = errors.New("not found")

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
		return repository.Episode{}, ErrMissingDescription
	}
	if req.File == nil {
		return repository.Episode{}, ErrMissingFile
	}

	data, ok := req.File.(io.ReadSeeker)
	if !ok {
		return repository.Episode{}, ErrFileNotSeekable
	}

	header := make([]byte, 512)
	n, readErr := data.Read(header)
	if readErr != nil && readErr != io.EOF {
		return repository.Episode{}, fmt.Errorf("failed to read file header: %w", readErr)
	}
	_ = n
	_, seekErr := data.Seek(0, io.SeekStart)
	if seekErr != nil {
		return repository.Episode{}, fmt.Errorf("failed to seek file: %w", seekErr)
	}

	mime, err := audio.DetectMIME(header, req.Filename)
	if err != nil {
		return repository.Episode{}, ErrUnsupportedMedia
	}

	meta, err := audio.Extract(data)
	if err != nil {
		return repository.Episode{}, fmt.Errorf("failed to extract metadata: %w", err)
	}
	_, seekErr = data.Seek(0, io.SeekStart)
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

	episodeUUID := uuid.New().String()
	episodeDir := filepath.Join(s.uploadDir, episodeUUID)
	err = os.MkdirAll(episodeDir, 0o755)
	if err != nil {
		return repository.Episode{}, fmt.Errorf("failed to create episode directory: %w", err)
	}

	ext := filepath.Ext(req.Filename)
	filePath := filepath.Join(episodeDir, episodeUUID+ext)
	file, err := os.Create(filePath)
	if err != nil {
		_ = os.RemoveAll(episodeDir)
		return repository.Episode{}, fmt.Errorf("failed to create file: %w", err)
	}

	written, err := io.Copy(file, data)
	if err != nil {
		_ = file.Close()
		_ = os.RemoveAll(episodeDir)
		return repository.Episode{}, fmt.Errorf("failed to write file: %w", err)
	}
	err = file.Close()
	if err != nil {
		_ = os.RemoveAll(episodeDir)
		return repository.Episode{}, fmt.Errorf("failed to close file: %w", err)
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
		_ = os.RemoveAll(episodeDir)
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

	episodeDir := filepath.Join(s.uploadDir, uuid)
	removeErr := os.RemoveAll(episodeDir)
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return fmt.Errorf("failed to remove episode directory: %w", removeErr)
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
