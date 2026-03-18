package audio

import (
	"errors"
	"io"

	"github.com/dhowden/tag"
)

type Meta struct {
	Title        string
	Artist       string
	DurationSecs int
}

func Extract(r io.ReadSeeker) (Meta, error) {
	meta := Meta{}

	metadata, err := tag.ReadFrom(r)
	if err != nil {
		if errors.Is(err, tag.ErrNoTagsFound) {
			return meta, nil
		}
		return meta, err
	}

	meta.Title = metadata.Title()
	meta.Artist = metadata.Artist()

	if meta.Title == "" {
		meta.Title = metadata.Album()
	}

	return meta, nil
}
