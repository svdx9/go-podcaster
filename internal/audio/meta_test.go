package audio

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestExtract(t *testing.T) { //nolint:paralleltest
	t.Run("handles file without tags", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "test-*.mp3")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		tmpName := tmp.Name()
		defer func() { _ = os.Remove(tmpName) }()
		_ = tmp.Close()

		f, err := os.Open(tmpName)
		if err != nil {
			t.Fatalf("failed to open temp file: %v", err)
		}
		defer func() { _ = f.Close() }()

		meta, err := Extract(f)
		if err != nil {
			if errors.Is(err, ErrFfprobeNotFound) {
				t.Skip("ffprobe not available on PATH")
			}
			t.Errorf("Extract() error = %v, want nil", err)
		}
		if meta.Title != "" {
			t.Errorf("Extract() title = %q, want empty", meta.Title)
		}
	})
}

func TestExtract_ErrFfprobeNotFound(t *testing.T) { //nolint:paralleltest

	origPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", origPath) }()

	_ = os.Setenv("PATH", "/nonexistent")

	_, err := exec.LookPath("ffprobe")
	if err == nil {
		t.Skip("could not simulate missing ffprobe")
	}

	tmp, err := os.CreateTemp("", "test-*.mp3")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	_ = tmp.Close()

	f, err := os.Open(tmpName)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer func() { _ = f.Close() }()

	_, err = Extract(f)
	if !errors.Is(err, ErrFfprobeNotFound) {
		t.Errorf("Extract() error = %v, want ErrFfprobeNotFound", err)
	}
}

func TestMetaDuration(t *testing.T) { //nolint:paralleltest
	m := Meta{ //nolint:exhaustruct
		DurationSecs: 125,
	}
	if m.DurationSecs != 125 {
		t.Errorf("DurationSecs = %d, want 125", m.DurationSecs)
	}
}
