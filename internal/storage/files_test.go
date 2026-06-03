package storage_test

import (
	"errors"
	"io"
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
			name:      "extension without . returns err",
			extension: "txt",
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseDir := t.TempDir()
			fs, err := storage.NewFileStore(baseDir)
			if err != nil {
				t.Fatalf("NewFileStore: %v", err)
			}
			id := uuid.New()
			got, _, err := fs.SaveFile(id, test.extension, test.src)
			if test.wantErr {
				if err == nil {
					t.Fatalf("wanted error, got nil")
				}
				wantPath := filepath.Join(baseDir, id.String()[:2], id.String()[2:]+test.extension)
				if _, statErr := os.Stat(wantPath); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("file should not exist in %s", wantPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("did not want error, got it")
			}
			wantPath := filepath.Join(id.String()[:2], id.String()[2:]+test.extension)
			if wantPath != got {
				t.Errorf("path %q, want %q", got, wantPath)
			}
			if _, statErr := os.Stat(filepath.Join(baseDir, got)); statErr != nil {
				t.Errorf("file should exist: %v", statErr)
			}
		})
	}
}
