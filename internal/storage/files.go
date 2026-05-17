package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// FileStore persists files on local disk under a UUID-prefixed directory layout.
// A document with ID "abcdef..." is stored as "<baseDir>/ab/cdef...<ext>",
// which avoids putting too many entries in one directory.
//
// All I/O is confined to baseDir through *os.Root. Relative paths that try to
// escape through ".." or symlinks return an error instead of accessing files
// outside the configured storage root.
type FileStore struct {
	root *os.Root
}

var validExt = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,16}$`)

// NewFileStore creates the base storage directory and opens it as an os.Root.
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, fmt.Errorf("open base dir as root: %w", err)
	}
	return &FileStore{root: root}, nil
}

// Close releases the underlying storage root.
func (f *FileStore) Close() error {
	return f.root.Close()
}

// SaveFile writes src to <baseDir>/<first 2 UUID chars>/<remaining UUID><ext>.
// It returns the relative storage path and the number of bytes written.
func (f *FileStore) SaveFile(id uuid.UUID, extension string, src io.Reader) (string, int64, error) {
	if !strings.HasPrefix(extension, ".") {
		return "", 0, errors.New("extension must start with '.'")
	}

	if !validExt.MatchString(extension) {
		return "", 0, errors.New("invalid extension")
	}

	idStr := id.String()
	subDir := idStr[:2]
	if err := f.root.MkdirAll(subDir, 0o750); err != nil {
		return "", 0, fmt.Errorf("create sub dir: %w", err)
	}

	relPath := filepath.Join(subDir, idStr[2:]+extension)

	dst, err := f.root.Create(relPath)
	if err != nil {
		return "", 0, fmt.Errorf("create file: %w", err)
	}

	size, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = f.root.Remove(relPath)
		return "", 0, fmt.Errorf("write file: %w", copyErr)
	}
	if closeErr != nil {
		_ = f.root.Remove(relPath)
		return "", 0, fmt.Errorf("close file: %w", closeErr)
	}

	return relPath, size, nil
}

// DeleteFile removes a file by its relative storage path.
func (f *FileStore) DeleteFile(relPath string) error {
	if err := f.root.Remove(relPath); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

// OpenFile opens a stored file by its relative storage path.
func (f *FileStore) OpenFile(path string) (io.ReadSeekCloser, error) {
	file, err := f.root.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("file does not exist: %w", err)
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	return file, nil
}

// Exists reports whether a relative storage path exists.
func (f *FileStore) Exists(relPath string) bool {
	_, err := f.root.Stat(relPath)
	return err == nil
}
