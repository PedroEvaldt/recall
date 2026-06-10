package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/spf13/viper"
)

// captured records the fields of an upload request observed by the fake
// server, so tests can assert what the CLI actually sent on the wire.
type captured struct {
	method      string
	path        string
	contentType string
	title       string
	tags        string
	authHeader  string
	fileContent []byte
}

func TestUpload_Success(t *testing.T) {
	resetState(t)

	tempFile := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(tempFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	srv, capturedFields := newUploadServer(t)
	defer srv.Close()

	t.Setenv("RECALL_SERVER", srv.URL)
	t.Setenv("RECALL_AUTH_TOKEN", "test-token")

	stdout := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetArgs([]string{"upload", tempFile, "-t", "My doc", "--tags", "go,devops"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected nil error; got %v", err)
	}
	if capturedFields.method != http.MethodPost {
		t.Errorf("method: expected %q; got %q", http.MethodPost, capturedFields.method)
	}
	if capturedFields.path != "/documents" {
		t.Errorf("path: expected %q; got %q", "/documents", capturedFields.path)
	}
	if capturedFields.title != "My doc" {
		t.Errorf("title: expected %q; got %q", "My doc", capturedFields.title)
	}
	if capturedFields.tags != "go,devops" {
		t.Errorf("tags: expected %q; got %q", "go,devops", capturedFields.tags)
	}
	if !bytes.Equal(capturedFields.fileContent, []byte("test content")) {
		t.Errorf("fileContent: expected %q; got %q", "test content", capturedFields.fileContent)
	}
	if capturedFields.authHeader != "Bearer test-token" {
		t.Errorf("authHeader: expected %q; got %q", "Bearer test-token", capturedFields.authHeader)
	}
	got := stdout.String()
	if !strings.Contains(got, "Uploaded document") {
		t.Errorf("stdout: expected to contain %q; got %q", "Uploaded document", got)
	}
	if !strings.Contains(got, "550e8400-e29b-41d4-a716-446655440000") {
		t.Errorf("stdout: expected to contain %q; got %q", "550e8400-e29b-41d4-a716-446655440000", got)
	}
}

func TestUpload_MissingFile(t *testing.T) {
	resetState(t)
	t.Setenv("RECALL_SERVER", "http://localhost:1")
	t.Setenv("RECALL_AUTH_TOKEN", "x")
	rootCmd.SetArgs([]string{"upload", "/not/exist/path/foo.md"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error containing %q; got nil", "no such file")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("expected error containing %q; got %v", "no such file", err)
	}
}

func TestUpload_Error(t *testing.T) {
	resetState(t)
	tempFile := filepath.Join(t.TempDir(), "foo.md")
	if err := os.WriteFile(tempFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()
	t.Setenv("RECALL_SERVER", srv.URL)
	t.Setenv("RECALL_AUTH_TOKEN", "x")
	rootCmd.SetArgs([]string{"upload", tempFile})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error containing %q; got nil", "post document")
	}
	if !strings.Contains(err.Error(), "post document") {
		t.Errorf("expected error containing %q; got %v", "post document", err)
	}
}

func TestUpload_EmptyURL(t *testing.T) {
	resetState(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RECALL_SERVER", "")
	t.Setenv("RECALL_AUTH_TOKEN", "")
	tempFile := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(tempFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	rootCmd.SetArgs([]string{"upload", tempFile})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error wrapping ErrEmptyURL; got nil")
	}
	if !errors.Is(err, client.ErrEmptyURL) {
		t.Errorf("expected error wrapping ErrEmptyURL; got %v", err)
	}
}

func TestDefaultTitle(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantTitle string
	}{
		{
			name:      "Normal markdown file",
			path:      "notes.md",
			wantTitle: "notes",
		},
		{
			name:      "Path to markdown file",
			path:      "/tmp/notes.md",
			wantTitle: "notes",
		},
		{
			name:      "Multiple extension",
			path:      "foo.tar.gz",
			wantTitle: "foo.tar",
		},
		{
			// FIXME: filepath.Ext treats the whole ".bashrc" as extension, so
			// the title ends up empty. Documented to flag silent regressions.
			name:      "Hidden file",
			path:      ".bashrc",
			wantTitle: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle := defaultTitle(tt.path)
			if gotTitle != tt.wantTitle {
				t.Errorf("expected %q; got %q", tt.wantTitle, gotTitle)
			}
		})
	}
}

// resetState clears package-level CLI state between tests so that flags or
// viper values set by one test do not leak into the next.
func resetState(t *testing.T) {
	t.Helper()
	viper.Reset()
	if err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server")); err != nil {
		t.Fatalf("could not bind server flag: %v", err)
	}
	if err := viper.BindPFlag("auth_token", rootCmd.PersistentFlags().Lookup("auth-token")); err != nil {
		t.Fatalf("could not bind auth-token flag: %v", err)
	}

	for _, name := range []string{"server", "auth-token"} {
		flag := rootCmd.PersistentFlags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag %q not found on rootCmd", name)
		}
		if err := flag.Value.Set(""); err != nil {
			t.Fatalf("could not reset flag %q value: %v", name, err)
		}
		flag.Changed = false
	}

	listAll = false
	limit = 0
	uploadTitle = ""
	uploadTags = nil
}

// newUploadServer starts an httptest server that parses an incoming multipart
// upload into capturedFields, so callers can assert what the CLI sent.
func newUploadServer(t *testing.T) (*httptest.Server, *captured) {
	t.Helper()
	capturedFields := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFields.method = r.Method
		capturedFields.path = r.URL.Path
		capturedFields.contentType = r.Header.Get("Content-Type")
		capturedFields.authHeader = r.Header.Get("Authorization")

		if err := r.ParseMultipartForm(20 << 20); err != nil { // #nosec G120 -- test fake handler, no attack surface
			t.Errorf("could not parse multipart form: %v", err)
			return
		}
		capturedFields.title = r.FormValue("title")
		capturedFields.tags = r.FormValue("tags")
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("could not get form file: %v", err)
			return
		}
		defer func() { _ = file.Close() }()
		capturedFields.fileContent, err = io.ReadAll(file)
		if err != nil {
			t.Errorf("could not read file body: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprintln(w, `{"id":"550e8400-e29b-41d4-a716-446655440000","title":"foo","filename":"foo.md"}`)
		if err != nil {
			t.Errorf("could not write id to header: %v", err)
		}
	}))
	return srv, capturedFields
}
