package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Episode struct {
	UUID         uuid.UUID
	Title        string
	Description  string
	Author       string
	PubDate      time.Time
	FilePath     string
	FileName     string
	FileSize     int64
	MimeType     string
	DurationSecs int
	CreatedAt    time.Time
}

type Repository interface {
	Insert(ctx context.Context, episode Episode) error
	GetByUUID(ctx context.Context, UUID uuid.UUID) (Episode, error)
	List(ctx context.Context, limit, offset int) ([]Episode, error)
	Delete(ctx context.Context, UUID uuid.UUID) error
	ListAll(ctx context.Context) ([]Episode, error)
	ListAllValid(ctx context.Context) ([]Episode, error)
	UpdateDuration(ctx context.Context, UUID uuid.UUID, durationSecs int) error
	ListPendingDuration(ctx context.Context) ([]Episode, error)
}
