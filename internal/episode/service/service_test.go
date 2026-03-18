package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/svdx9/go-podcaster/internal/episode/repository"
)

var (
	errTestRead = errors.New("read error")
)

type mockRepository struct {
	episodes []repository.Episode
	err      error
}

func (m *mockRepository) Insert(ctx context.Context, episode repository.Episode) error {
	m.episodes = append(m.episodes, episode)
	return m.err
}

func (m *mockRepository) GetByUUID(ctx context.Context, uuid string) (repository.Episode, error) {
	for _, ep := range m.episodes {
		if ep.UUID == uuid {
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

func (m *mockRepository) Delete(ctx context.Context, uuid string) error {
	for i, ep := range m.episodes {
		if ep.UUID == uuid {
			m.episodes = append(m.episodes[:i], m.episodes[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockRepository) ListAll(ctx context.Context) ([]repository.Episode, error) {
	return m.episodes, m.err
}

type nonSeeker struct {
	data    []byte
	pos     int
	readErr bool
}

func (n *nonSeeker) Read(p []byte) (int, error) {
	if n.pos >= len(n.data) {
		return 0, io.EOF
	}
	n.pos++
	return copy(p, n.data[n.pos-1:n.pos]), nil
}

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
		return 0, nil
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := New(&mockRepository{}, "/tmp/uploads")
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
			{UUID: "1", Title: "Ep1", Description: "", Author: "", PubDate: time.Time{}, FilePath: "", FileName: "", FileSize: 0, MimeType: "", DurationSecs: 0, CreatedAt: time.Now()},
			{UUID: "2", Title: "Ep2", Description: "", Author: "", PubDate: time.Time{}, FilePath: "", FileName: "", FileSize: 0, MimeType: "", DurationSecs: 0, CreatedAt: time.Now()},
			{UUID: "3", Title: "Ep3", Description: "", Author: "", PubDate: time.Time{}, FilePath: "", FileName: "", FileSize: 0, MimeType: "", DurationSecs: 0, CreatedAt: time.Now()},
		},
		err: nil,
	}
	svc := New(repo, "/tmp/uploads")

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
