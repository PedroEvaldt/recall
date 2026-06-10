package storage_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/google/uuid"
)

func TestSaveFile(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name      string
		extension string
		src       io.Reader
		wantErr   bool
	}{
		{
			// Verify that the expected UUID-prefixed directory is created.
			name:      "create file in the correct path",
			extension: ".txt",
			src:       strings.NewReader("content"),
			wantErr:   false,
		},
		{
			name:      "extension without .",
			extension: "txt",
			src:       strings.NewReader("content"),
			wantErr:   true,
		},
		{
			name:      "invalid extension",
			extension: ".foo -  %",
			src:       strings.NewReader("content"),
			wantErr:   true,
		},
		{
			name:      "error in copy removes partial file",
			extension: ".txt",
			src:       iotest.ErrReader(boom),
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := storage.NewFileStore(baseDir)
			if err != nil {
				t.Fatalf("NewFileStore: %v", err)
			}
			id := uuid.New()
			got, size, err := store.SaveFile(id, tt.extension, tt.src)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("wanted error, got nil")
				}
				wantPath := filepath.Join(baseDir, id.String()[:2], id.String()[2:]+tt.extension)
				if _, statErr := os.Stat(wantPath); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("file should not exist in %s", wantPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("did not want error, got it")
			}
			wantPath := filepath.Join(id.String()[:2], id.String()[2:]+tt.extension)
			if wantPath != got {
				t.Errorf("path %q, want %q", got, wantPath)
			}
			if _, statErr := os.Stat(filepath.Join(baseDir, got)); statErr != nil {
				t.Errorf("file should exist: %v", statErr)
			}
			if size != int64(len("content")) {
				t.Errorf("expected size: %v; got %v", int64(len("content")), size)
			}
		})
	}
}

func TestSaveFileOverwrites(t *testing.T) {
	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	id := uuid.New()
	if _, _, err := store.SaveFile(id, ".txt", strings.NewReader("longer original content")); err != nil {
		t.Fatalf("first save: %v", err)
	}
	path, size, err := store.SaveFile(id, ".txt", strings.NewReader("hi"))
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if size != 2 {
		t.Errorf("expected size: 2; got %v", size)
	}

	f, err := store.OpenFile(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer func() { _ = f.Close() }()
	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("expected content hi; got %v", string(got))
	}
}

func TestLifecycle(t *testing.T) {
	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	id := uuid.New()
	content := []byte("test content")
	idStr := id.String()
	relPath := filepath.Join(idStr[:2], idStr[2:]+".txt")

	// 1. Exists on a path that has never been written returns false.
	if store.Exists(relPath) {
		t.Errorf("Exists should be false before SaveFile")
	}

	// 2. OpenFile on a missing path wraps fs.ErrNotExist.
	if _, err := store.OpenFile("zz/does-not-exist.txt"); err == nil || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("OpenFile should wrap fs.ErrNotExist, got %v", err)
	}

	// 3. DeleteFile on a missing path returns an error.
	if err := store.DeleteFile("zz/does-not-exist.txt"); err == nil {
		t.Errorf("DeleteFile should fail for missing path")
	}

	// 4. SaveFile then Exists returns true.
	path, _, err := store.SaveFile(id, ".txt", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	if !store.Exists(path) {
		t.Errorf("Exists should be true after SaveFile")
	}

	// 5. OpenFile returns the original content.
	f, err := store.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	got, readErr := io.ReadAll(f)
	err = f.Close()
	if err != nil {
		t.Fatalf("close file: %v", err)
	}
	if readErr != nil {
		t.Fatalf("ReadAll: %v", readErr)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}

	// 6. DeleteFile then Exists returns false.
	if err := store.DeleteFile(path); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if store.Exists(path) {
		t.Errorf("Exists should be false after DeleteFile")
	}
}

func TestClose(t *testing.T) {
	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	id := uuid.New()
	path, _, err := store.SaveFile(id, ".txt", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("save file: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := store.OpenFile(path); err == nil {
		t.Errorf("OpenFile should fail after Close, got nil error")
	}
	if err := store.DeleteFile(path); err == nil {
		t.Errorf("DeleteFile should fail after Close, got nil error")
	}
	if store.Exists(path) {
		t.Errorf("Exists should return false after Close")
	}
}

func TestNewFileStore(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "nested", "sub")
	store, err := storage.NewFileStore(baseDir)
	if err != nil {
		t.Fatalf("NewFileStore")
	}
	defer func() { _ = store.Close() }()
	if info, err := os.Stat(baseDir); err != nil || !info.IsDir() {
		t.Errorf("baseDir should be created as directory")
	}

	file := filepath.Join(t.TempDir(), "imafile")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := storage.NewFileStore(file); err == nil {
		t.Errorf("expected error when baseDir is an existing file")
	}
}
