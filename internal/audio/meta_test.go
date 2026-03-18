package audio

import (
	"os"
	"testing"
)

func TestExtract(t *testing.T) {
	t.Parallel()
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
			t.Errorf("Extract() error = %v, want nil", err)
		}
		if meta.Title != "" {
			t.Errorf("Extract() title = %q, want empty", meta.Title)
		}
	})
}

func TestMetaDuration(t *testing.T) {
	t.Parallel()
	m := Meta{
		DurationSecs: 125,
	}
	if m.DurationSecs != 125 {
		t.Errorf("DurationSecs = %d, want 125", m.DurationSecs)
	}
}
