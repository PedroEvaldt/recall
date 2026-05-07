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

// FileStore persiste arquivos no disco local em uma estrutura particionada por
// prefixo do UUID (ex: <baseDir>/ab/cdef.../file.pdf), evitando diretórios com
// dezenas de milhares de entradas. As operações de IO ficam confinadas ao
// baseDir via *os.Root: paths com ".." ou symlinks apontando pra fora retornam
// erro em vez de escapar.
type FileStore struct {
	root *os.Root
}

var validExt = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,16}$`)

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

func (f *FileStore) Close() error {
	return f.root.Close()
}

// SaveFile escreve o conteúdo em <baseDir>/<2 chars do UUID>/<resto>.<ext> e
// devolve o caminho relativo ao baseDir e o tamanho em bytes gravado.
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

func (f *FileStore) DeleteFile(relPath string) error {
	if err := f.root.Remove(relPath); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

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

func (f *FileStore) Exists(relPath string) bool {
	_, err := f.root.Stat(relPath)
	return err == nil
}
