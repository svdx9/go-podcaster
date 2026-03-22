package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/dhowden/tag"
)

// ErrFfprobeNotFound is returned when ffprobe is not available on PATH.
var ErrFfprobeNotFound = errors.New("ffprobe not found on PATH")

type Meta struct {
	Title        string
	Artist       string
	DurationSecs int
}

type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func Extract(r io.ReadSeeker) (Meta, error) {
	meta := Meta{}

	metadata, err := ReadTags(r)
	if err != nil {
		return meta, err
	}
	meta.Title = metadata.Title
	meta.Artist = metadata.Artist

	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return meta, fmt.Errorf("failed to seek file for duration extraction: %w", err)
	}

	secs, err := ProbeDuration(r)
	if err != nil {
		return meta, fmt.Errorf("failed to extract duration: %w", err)
	}
	meta.DurationSecs = secs

	return meta, nil
}

func ReadTags(r io.ReadSeeker) (Tags, error) {
	result := Tags{}

	metadata, err := tag.ReadFrom(r)
	if err != nil {
		if errors.Is(err, tag.ErrNoTagsFound) || errors.Is(err, io.EOF) {
			return result, nil
		}
		return result, err
	}

	result.Title = metadata.Title()
	result.Artist = metadata.Artist()

	if result.Title == "" {
		result.Title = metadata.Album()
	}

	return result, nil
}

func ProbeDuration(r io.Reader) (int, error) {
	return ffprobeDuration(r)
}

type Tags struct {
	Title  string
	Artist string
}

// ffprobeDuration writes r to a temp file, runs ffprobe, and returns duration in seconds.
func ffprobeDuration(r io.Reader) (int, error) {
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, ErrFfprobeNotFound
	}

	tmp, err := os.CreateTemp("", "audio-*.tmp")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	_, err = io.Copy(tmp, r)
	if err != nil {
		_ = tmp.Close()
		return 0, fmt.Errorf("failed to write temp file: %w", err)
	}
	err = tmp.Close()
	if err != nil {
		return 0, fmt.Errorf("failed to close temp file: %w", err)
	}

	ctx := context.Background()
	out, err := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		tmpName,
	).Output()
if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result ffprobeOutput
	err = json.Unmarshal(out, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if result.Format.Duration == "" {
		return 0, nil
	}

	f, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration %q: %w", result.Format.Duration, err)
	}

	return int(f), nil
}
