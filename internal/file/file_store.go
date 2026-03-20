package file

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var (
	ErrFileNotFound  = errors.New("file does not exist")
	ErrInitFailed    = errors.New("failed to initialize file store")
	ErrFileCreate    = errors.New("failed to create file")
	ErrFileClose     = errors.New("failed to close file")
	ErrFileDelete    = errors.New("failed to delete file")
	ErrTmpFileCreate = errors.New("failed to create temp file")
)

// Store abstracts file persistence so the service has no direct
// dependency on the local filesystem.
type Store interface {
	// Save writes r to storage keyed by uuid and returns the stored
	// path and the number of bytes written.
	Save(r io.Reader) (UUID uuid.UUID, written int64, err error)
	// Delete removes all stored files for uuid.
	Delete(id uuid.UUID) error

	// will read the file at the given path and call fn
	ReadSeekFile(id uuid.UUID, fn func(io.ReadSeeker) error) error
}

// LocalStore is the production Store backed by the local filesystem.
type LocalStore struct {
	uploadDir string
}

func NewStore(uploadDir string) *LocalStore {
	return &LocalStore{uploadDir: uploadDir}
}

func (l *LocalStore) Init() error {
	err := os.MkdirAll(l.uploadDir, 0o755)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitFailed, err)
	}
	return nil
}

func (l *LocalStore) Save(r io.Reader) (uuid.UUID, int64, error) {
	hash := sha256.New()
	// temp file in same directory (ensures same filesystem for atomic move)
	tmpFile, err := os.CreateTemp(l.uploadDir, "tmp-*")
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w: %w", ErrTmpFileCreate, err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name()) // cleanup if move fails
	}()

	writer := io.MultiWriter(hash, tmpFile)
	writtenBytes, err := io.Copy(writer, r)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w: %w", ErrFileCreate, err)
	}
	err = tmpFile.Close()
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w: %w", ErrFileClose, err)
	}

	// compute hash
	new, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w: %w", ErrFileCreate, err)
	}
	uuid := uuid.NewSHA1(new, hash.Sum(nil))

	uuidPath := filepath.Join(l.uploadDir, uuid.String())
	err = os.Rename(tmpFile.Name(), uuidPath)
	if err != nil {
		return uuid, writtenBytes, fmt.Errorf("%w: %w", ErrFileCreate, err)
	}
	return uuid, writtenBytes, nil
}

func (l *LocalStore) Delete(uuid uuid.UUID) error {
	episodeDir := filepath.Join(l.uploadDir, uuid.String())
	err := os.RemoveAll(episodeDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("%w: %w", ErrFileDelete, err)
	}
	return nil
}

func (s *LocalStore) ReadSeekFile(u uuid.UUID, fn func(io.ReadSeeker) error) error {
	filePath := filepath.Join(s.uploadDir, u.String())
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %w", ErrFileNotFound, err)
		}
		return err
	}
	defer func() { _ = f.Close() }()
	return fn(f)
}
