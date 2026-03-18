package repository

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Episode struct {
	UUID         string
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
	GetByUUID(ctx context.Context, uuid string) (Episode, error)
	List(ctx context.Context, limit, offset int) ([]Episode, error)
	Delete(ctx context.Context, uuid string) error
	ListAll(ctx context.Context) ([]Episode, error)
}
