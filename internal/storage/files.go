package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// FileStore persiste arquivos no disco local em uma estrutura particionada por
// prefixo do UUID (ex: <baseDir>/ab/cdef.../file.pdf), evitando diretórios com
// dezenas de milhares de entradas.
type FileStore struct {
	baseDir string
}

func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

// SaveFile escreve o conteúdo em <baseDir>/<2 chars do UUID>/<resto>.<ext> e
// devolve o caminho relativo ao baseDir e o tamanho em bytes gravado.
func (f *FileStore) SaveFile(id uuid.UUID, extension string, src io.Reader) (string, int64, error) {
	if !strings.HasPrefix(extension, ".") {
		return "", 0, errors.New("extension must start with '.'")
	}

	idStr := id.String()
	subDir := filepath.Join(f.baseDir, idStr[:2])
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		return "", 0, fmt.Errorf("create sub dir: %w", err)
	}

	relPath := filepath.Join(idStr[:2], idStr[2:]+extension)
	absPath := filepath.Join(f.baseDir, relPath)

	dst, err := os.Create(absPath)
	if err != nil {
		return "", 0, fmt.Errorf("create file: %w", err)
	}

	size, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(absPath)
		return "", 0, fmt.Errorf("write file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(absPath)
		return "", 0, fmt.Errorf("close file: %w", closeErr)
	}

	return relPath, size, nil
}

func (f *FileStore) DeleteFile(relPath string) error {
	if err := os.Remove(filepath.Join(f.baseDir, relPath)); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
