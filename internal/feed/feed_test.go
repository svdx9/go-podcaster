package feed

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
	"github.com/svdx9/go-podcaster/internal/episode/repository/repositorytest"
)

func TestRender(t *testing.T) {
	t.Parallel()
	pubDate := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2024, 1, 15, 10, 5, 0, 0, time.UTC)

	episodeUUID := uuid.Must(uuid.NewV7())

	repo := &repositorytest.MockRepository{
		Episodes: []repository.Episode{
			{
				UUID:         episodeUUID,
				Title:        "Test Episode",
				Description:  "A test episode description",
				Author:       "Test Author",
				PubDate:      pubDate,
				FilePath:     "/uploads/test-uuid-1/test.mp3",
				FileName:     "test.mp3",
				FileSize:     1024,
				MimeType:     "audio/mpeg",
				DurationSecs: 3600,
				CreatedAt:    createdAt,
			},
		},
		Err: nil,
	}

	g := New(repo, "https://example.com", "Test Podcast", "A test podcast", "Podcast Author", "en-us", "Technology", "")

	var buf bytes.Buffer
	err := g.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	xml := buf.String()

	if !strings.Contains(xml, "<title>Test Podcast</title>") {
		t.Error("Render() should contain podcast title")
	}
	if !strings.Contains(xml, "<title>Test Episode</title>") {
		t.Error("Render() should contain episode title")
	}
	if !strings.Contains(xml, "<enclosure") {
		t.Error("Render() should contain enclosure")
	}
	expectedURL := "https://example.com/files/" + episodeUUID.String()
	if !strings.Contains(xml, expectedURL) {
		t.Errorf("Render() should contain enclosure URL %s", expectedURL)
	}
}

func TestRenderEmpty(t *testing.T) {
	t.Parallel()
	repo := &repositorytest.MockRepository{Episodes: []repository.Episode{}, Err: nil}

	g := New(repo, "https://example.com", "Empty Podcast", "No episodes", "Author", "en-us", "", "")

	var buf bytes.Buffer
	err := g.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	xml := buf.String()
	if !strings.Contains(xml, "<title>Empty Podcast</title>") {
		t.Error("Render() should contain podcast title even with no episodes")
	}
	if strings.Contains(xml, "<item>") {
		t.Error("Render() should not contain items when episode list is empty")
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		secs int
		want string
	}{
		{0, "0:00"},
		{30, "0:30"},
		{60, "1:00"},
		{90, "1:30"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{7200, "2:00:00"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.secs)
		if got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.secs, got, tt.want)
		}
	}
}
