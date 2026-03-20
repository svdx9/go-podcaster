package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
	"github.com/svdx9/go-podcaster/internal/file"
)

var errTestRead = errors.New("read error")

type mockRepository struct {
	episodes []repository.Episode
	err      error
}

func (m *mockRepository) Insert(ctx context.Context, episode repository.Episode) error {
	m.episodes = append(m.episodes, episode)
	return m.err
}

func (m *mockRepository) GetByUUID(ctx context.Context, UUID uuid.UUID) (repository.Episode, error) {
	for _, ep := range m.episodes {
		if ep.UUID == UUID {
			return ep, nil
		}
	}
	return repository.Episode{}, repository.ErrNotFound
}

func (m *mockRepository) List(ctx context.Context, limit, offset int) ([]repository.Episode, error) {
	if len(m.episodes) <= offset {
		return []repository.Episode{}, nil
	}
	end := offset + limit
	if end > len(m.episodes) {
		end = len(m.episodes)
	}
	return m.episodes[offset:end], m.err
}

func (m *mockRepository) Delete(ctx context.Context, UUID uuid.UUID) error {
	for i, ep := range m.episodes {
		if ep.UUID == UUID {
			m.episodes = append(m.episodes[:i], m.episodes[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockRepository) ListAll(ctx context.Context) ([]repository.Episode, error) {
	return m.episodes, m.err
}

func (m *mockRepository) UpdateDuration(ctx context.Context, UUID uuid.UUID, durationSecs int) error {
	return m.err
}

func (m *mockRepository) ListPendingDuration(ctx context.Context) ([]repository.Episode, error) {
	return nil, m.err
}

type memFileStore struct{}

func (m *memFileStore) Save(r io.Reader) (uuid.UUID, int64, error) {
	hash := sha256.New()
	writer := io.MultiWriter(hash, io.Discard)
	writtenBytes, err := io.Copy(writer, r)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w: %w", file.ErrFileCreate, err)
	}
	new, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w: %w", file.ErrFileCreate, err)
	}
	uuid := uuid.NewSHA1(new, hash.Sum(nil))
	return uuid, writtenBytes, nil
}

func (m *memFileStore) Delete(uuid uuid.UUID) error                                     { return nil }
func (m *memFileStore) ReadSeekFile(uuid uuid.UUID, fn func(io.ReadSeeker) error) error { return nil }

type mockReadSeeker struct {
	data    []byte
	pos     int
	readErr bool
}

func (m *mockReadSeeker) Read(p []byte) (n int, err error) {
	if m.readErr {
		return 0, errTestRead
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		m.pos = int(offset)
	case 1:
		m.pos += int(offset)
	case 2:
		m.pos = len(m.data) + int(offset)
	}
	return int64(m.pos), nil
}

func TestUploadValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     UploadRequest
		wantErr error
	}{
		{
			name: "missing description",
			req: UploadRequest{
				Title:       "Test",
				Description: "",
				Author:      "",
				PubDate:     "",
				File:        &mockReadSeeker{data: []byte{0xFF, 0xFB}, pos: 0, readErr: false},
				Filename:    "test.mp3",
				FileSize:    0,
			},
			wantErr: ErrMissingDescription,
		},
		{
			name: "missing file",
			req: UploadRequest{
				Title:       "Test",
				Description: "Test description",
				Author:      "",
				PubDate:     "",
				File:        nil,
				Filename:    "test.mp3",
				FileSize:    0,
			},
			wantErr: ErrMissingFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := New(slog.Default(), &mockRepository{}, &memFileStore{})
			_, err := svc.Upload(context.Background(), tt.req)
			if err == nil {
				t.Fatal("Upload() expected error, got nil")
			}
			if err.Error() != tt.wantErr.Error() {
				t.Errorf("Upload() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestListPagination(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		episodes: []repository.Episode{
			{UUID: uuid.UUID{1}, Title: "Ep1", Description: "", Author: "", PubDate: time.Time{}, FilePath: "", FileName: "", FileSize: 0, MimeType: "", DurationSecs: 0, CreatedAt: time.Now()},
			{UUID: uuid.UUID{2}, Title: "Ep2", Description: "", Author: "", PubDate: time.Time{}, FilePath: "", FileName: "", FileSize: 0, MimeType: "", DurationSecs: 0, CreatedAt: time.Now()},
			{UUID: uuid.UUID{3}, Title: "Ep3", Description: "", Author: "", PubDate: time.Time{}, FilePath: "", FileName: "", FileSize: 0, MimeType: "", DurationSecs: 0, CreatedAt: time.Now()},
		},
		err: nil,
	}
	svc := New(slog.Default(), repo, &memFileStore{})

	t.Run("default limit", func(t *testing.T) {
		t.Parallel()
		episodes, err := svc.List(context.Background(), 0, 0)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(episodes) != 3 {
			t.Errorf("List() returned %d episodes, want 3", len(episodes))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		t.Parallel()
		episodes, err := svc.List(context.Background(), 2, 0)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(episodes) != 2 {
			t.Errorf("List() returned %d episodes, want 2", len(episodes))
		}
	})

	t.Run("with offset", func(t *testing.T) {
		t.Parallel()
		episodes, err := svc.List(context.Background(), 10, 1)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(episodes) != 2 {
			t.Errorf("List() returned %d episodes, want 2", len(episodes))
		}
	})

	t.Run("negative offset becomes zero", func(t *testing.T) {
		t.Parallel()
		episodes, err := svc.List(context.Background(), 10, -5)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(episodes) != 3 {
			t.Errorf("List() returned %d episodes, want 3", len(episodes))
		}
	})
}
