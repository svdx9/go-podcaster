package audio

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

var AllowedMIMETypes = map[string]struct{}{
	"audio/mpeg":   {},
	"audio/x-m4a":  {},
	"audio/mp4":    {},
	"audio/aac":    {},
	"audio/x-aiff": {},
	"audio/aiff":   {},
	"audio/wav":    {},
	"audio/x-wav":  {},
}

var ErrUnsupportedMIMEType = errors.New("unsupported MIME type")

func DetectMIME(data []byte, filename string) (string, error) {
	detected := http.DetectContentType(data)

	ext := strings.ToLower(filepath.Ext(filename))
	extMap := map[string]string{
		".mp3":  "audio/mpeg",
		".m4a":  "audio/x-m4a",
		".mp4":  "audio/mp4",
		".aac":  "audio/aac",
		".aiff": "audio/x-aiff",
		".aif":  "audio/aiff",
		".wav":  "audio/x-wav",
	}

	if extMIME, ok := extMap[ext]; ok {
		if _, allowed := AllowedMIMETypes[extMIME]; allowed {
			return extMIME, nil
		}
		return "", fmt.Errorf("%w: %s", ErrUnsupportedMIMEType, extMIME)
	}

	// TODO: MIME detection relies on file extension. http.DetectContentType does not
	// recognise audio formats and returns application/octet-stream for them.
	// A more robust check would use a dedicated audio sniffing library.
	if _, allowed := AllowedMIMETypes[detected]; allowed {
		return detected, nil
	}

	return "", fmt.Errorf("%w: %s", ErrUnsupportedMIMEType, detected)
}
