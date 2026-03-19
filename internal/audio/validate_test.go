package audio

import (
	"testing"
)

func TestDetectMIME(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     []byte
		filename string
		want     string
		wantErr  bool
	}{
		{
			name:     "mp3 file by extension",
			data:     []byte{},
			filename: "test.mp3",
			want:     "audio/mpeg",
			wantErr:  false,
		},
		{
			name:     "m4a file by extension",
			data:     []byte{},
			filename: "test.m4a",
			want:     "audio/x-m4a",
			wantErr:  false,
		},
		{
			name:     "wav file by extension",
			data:     []byte{},
			filename: "test.wav",
			want:     "audio/x-wav",
			wantErr:  false,
		},
		{
			name:     "aac file by extension",
			data:     []byte{},
			filename: "test.aac",
			want:     "audio/aac",
			wantErr:  false,
		},
		{
			name:     "aiff file by extension",
			data:     []byte{},
			filename: "test.aiff",
			want:     "audio/x-aiff",
			wantErr:  false,
		},
		{
			name:     "unsupported extension",
			data:     []byte{},
			filename: "test.pdf",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "unsupported magic bytes",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			filename: "test.unknown",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := DetectMIME(tt.data, tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectMIME() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DetectMIME() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllowedMIMETypes(t *testing.T) {
	t.Parallel()
	expected := []string{
		"audio/mpeg",
		"audio/x-m4a",
		"audio/mp4",
		"audio/aac",
		"audio/x-aiff",
		"audio/wav",
		"audio/x-wav",
	}

	for _, mime := range expected {
		if _, ok := AllowedMIMETypes[mime]; !ok {
			t.Errorf("Expected %q to be in AllowedMIMETypes", mime)
		}
	}
}
