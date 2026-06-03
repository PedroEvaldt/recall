package client_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/google/uuid"
)

// newClient is a helper that builds a Client pointing at srv. Fails the test on error.
func newClient(t *testing.T, srvURL string) *client.Client {
	t.Helper()
	c, err := client.New(srvURL, 2*time.Second, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	return c
}

func TestNewClient(t *testing.T) {
	timeout := 100 * time.Second
	tests := []struct {
		name    string
		baseURL string
		wantErr error
	}{
		{
			name:    "url empty",
			baseURL: "",
			wantErr: client.ErrEmptyURL,
		},
		{
			name:    "invalid url",
			baseURL: "http://invalidurlbla  bla:8080",
			wantErr: client.ErrInvalidURL,
		},
		{
			name:    "url without scheme",
			baseURL: "//localhost:8080",
			wantErr: client.ErrMissingPart,
		},
		{
			name:    "url without host",
			baseURL: "http://",
			wantErr: client.ErrMissingPart,
		},
		{
			name:    "url correct",
			baseURL: "http://localhost:8080",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := client.New(tt.baseURL, timeout, "test-token")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New(%q) err = %v, want %v", tt.baseURL, err, tt.wantErr)
			}
			if (tt.wantErr == nil) && c == nil {
				t.Fatalf("New() returned nil client without an error")
			}
		})
	}
}

// TestListDocuments_Success covers the happy path plus capture of the request
// shape: query parameters and Authorization header.
func TestListDocuments_Success(t *testing.T) {
	var (
		gotPath   string
		gotQuery  string
		gotAuth   string
		gotMethod string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Two documents with distinguishable fields.
		w.Write([]byte(`[
			{"id":"11111111-1111-1111-1111-111111111111","title":"first","mime_type":"text/plain","filename":"a.txt","tags":["x"],"created_at":"2026-01-01T00:00:00Z"},
			{"id":"22222222-2222-2222-2222-222222222222","title":"second","mime_type":"text/markdown","filename":"b.md","tags":[],"created_at":"2026-01-02T00:00:00Z"}
		]`))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL)
	docs, err := c.ListDocuments(context.Background(), "foo", "5")
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}

	// Response shape.
	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2", len(docs))
	}
	if docs[0].Title != "first" || docs[1].Title != "second" {
		t.Errorf("titles = %q, %q; want first, second", docs[0].Title, docs[1].Title)
	}

	// Request shape captured by the handler.
	if gotPath != "/documents" {
		t.Errorf("path = %q, want /documents", gotPath)
	}
	if gotQuery != "limit=5&q=foo" && gotQuery != "q=foo&limit=5" {
		t.Errorf("query = %q, want q=foo&limit=5 (any order)", gotQuery)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotMethod != http.MethodGet {
		t.Errorf("Method = %q, want %q", gotMethod, http.MethodGet)
	}
}

// TestListDocuments_Errors covers all the non-200 / decode / cancellation paths.
func TestListDocuments_Errors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		cancelCtx  bool
		wantSubstr string
	}{
		{
			name:       "401 with error body",
			status:     http.StatusUnauthorized,
			body:       `{"error":"unauthorized"}`,
			wantSubstr: "unauthorized",
		},
		{
			name:       "500 without body",
			status:     http.StatusInternalServerError,
			body:       "",
			wantSubstr: "server returned 500",
		},
		{
			name:       "200 with invalid JSON",
			status:     http.StatusOK,
			body:       `{not json`,
			wantSubstr: "decode response",
		},
		{
			name:       "context canceled",
			cancelCtx:  true,
			wantSubstr: "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					w.Write([]byte(tt.body))
				}
			}))
			defer srv.Close()

			c := newClient(t, srv.URL)
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelCtx {
				cancel() // cancel before the call so Do returns immediately.
			} else {
				defer cancel()
			}

			_, err := c.ListDocuments(ctx, "", "")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("err = %v; want substring %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestGetContent_Success(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotMethod string
	)
	id := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()
	c := newClient(t, srv.URL)
	result, err := c.GetContent(context.Background(), id)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}
	defer result.Close()

	wantPath := "/documents/" + id.String() + "/content"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotMethod != http.MethodGet {
		t.Errorf("Method = %q, want %q", gotMethod, http.MethodGet)
	}
	content, err := io.ReadAll(result)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(content, []byte("hello world")) {
		t.Errorf("content = %q, want %q", content, "hello world")
	}
}

func TestGetContent_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := newClient(t, srv.URL)
	if _, err := c.GetContent(context.Background(), uuid.New()); !errors.Is(err, client.ErrNotFound) {
		t.Errorf("err = %v, want error wrapping client.ErrNotFound", err)
	}
}

// TestGetContent_Errors covers the non-404 error branches: status with error
// body, status without body, and context cancellation.
func TestGetContent_Errors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		cancelCtx  bool
		wantSubstr string
	}{
		{
			name:       "401 with error body",
			status:     http.StatusUnauthorized,
			body:       `{"error":"unauthorized"}`,
			wantSubstr: "unauthorized",
		},
		{
			name:       "500 without body",
			status:     http.StatusInternalServerError,
			body:       "",
			wantSubstr: "server returned 500",
		},
		{
			name:       "context canceled",
			cancelCtx:  true,
			wantSubstr: "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					w.Write([]byte(tt.body))
				}
			}))
			defer srv.Close()

			c := newClient(t, srv.URL)
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelCtx {
				cancel()
			} else {
				defer cancel()
			}

			_, err := c.GetContent(ctx, uuid.New())
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("err = %v; want substring %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestGetMeta_Success(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotMethod string
	)
	id := uuid.New()
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respMeta := api.MetaResponse{
			Title:     "testTitle",
			MimeType:  "text/markdown",
			Tags:      []string{"tag1", "tag2"},
			CreatedAt: createdAt,
		}
		if err := json.NewEncoder(w).Encode(respMeta); err != nil {
			t.Errorf("Encode response: %v", err)
		}
	}))
	defer srv.Close()
	c := newClient(t, srv.URL)
	meta, err := c.GetMeta(context.Background(), id)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}

	wantPath := "/documents/" + id.String()
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotMethod != http.MethodGet {
		t.Errorf("Method = %q, want %q", gotMethod, http.MethodGet)
	}
	if meta.Title != "testTitle" {
		t.Errorf("Title = %q, want testTitle", meta.Title)
	}
	if meta.MimeType != "text/markdown" {
		t.Errorf("MimeType = %q, want text/markdown", meta.MimeType)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "tag1" || meta.Tags[1] != "tag2" {
		t.Errorf("Tags = %v, want [tag1 tag2]", meta.Tags)
	}
	if !meta.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v", meta.CreatedAt, createdAt)
	}
}

func TestGetMeta_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := newClient(t, srv.URL)
	if _, err := c.GetMeta(context.Background(), uuid.New()); !errors.Is(err, client.ErrNotFound) {
		t.Errorf("err = %v, want error wrapping client.ErrNotFound", err)
	}
}

func TestGetMeta_Errors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		cancelCtx  bool
		wantSubstr string
	}{
		{
			name:       "401 with error body",
			status:     http.StatusUnauthorized,
			body:       `{"error":"unauthorized"}`,
			wantSubstr: "unauthorized",
		},
		{
			name:       "500 without body",
			status:     http.StatusInternalServerError,
			body:       "",
			wantSubstr: "server returned 500",
		},
		{
			name:       "200 with invalid JSON",
			status:     http.StatusOK,
			body:       `{not json`,
			wantSubstr: "decode response",
		},
		{
			name:       "context canceled",
			cancelCtx:  true,
			wantSubstr: "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					w.Write([]byte(tt.body))
				}
			}))
			defer srv.Close()

			c := newClient(t, srv.URL)
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelCtx {
				cancel()
			} else {
				defer cancel()
			}

			_, err := c.GetMeta(ctx, uuid.New())
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("err = %v; want substring %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestPostDocument_Success(t *testing.T) {
	var (
		gotPath     string
		gotAuth     string
		gotMethod   string
		gotTitle    string
		gotTags     string
		gotFilename string
		gotFileBody []byte
	)
	serverResponseID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			return
		}
		gotTitle = r.FormValue("title")
		gotTags = r.FormValue("tags")
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			return
		}
		defer file.Close()
		gotFilename = header.Filename
		gotFileBody, _ = io.ReadAll(file)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		respDoc := api.DocumentResponse{
			ID:        serverResponseID,
			Title:     "testTitle",
			MimeType:  "text/markdown",
			Filename:  "testFilename.md",
			Tags:      []string{"tag1", "tag2"},
			CreatedAt: time.Now().UTC(),
		}
		err = json.NewEncoder(w).Encode(respDoc)
		if err != nil {
			t.Errorf("Encode response: %v", err)
			return
		}
	}))
	defer srv.Close()
	c := newClient(t, srv.URL)
	fileContent := []byte("test file content")
	doc, err := c.PostDocument(context.Background(), "testTitle", "testFilename.md", bytes.NewReader(fileContent), []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("PostDocument: %v", err)
	}
	if gotPath != "/documents" {
		t.Errorf("path = %q, want /documents", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotMethod != http.MethodPost {
		t.Errorf("Method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotTitle != "testTitle" {
		t.Errorf("Title = %q, want testTitle", gotTitle)
	}
	if gotTags != "tag1,tag2" {
		t.Errorf("Tags = %q, want tag1,tag2", gotTags)
	}
	if gotFilename != "testFilename.md" {
		t.Errorf("Filename = %q, want testFilename.md", gotFilename)
	}
	if !bytes.Equal(fileContent, gotFileBody) {
		t.Errorf("FileBody = %q, want %q", gotFileBody, fileContent)
	}
	if doc.ID != serverResponseID {
		t.Errorf("doc.ID = %v, want %v", doc.ID, serverResponseID)
	}
	if doc.Title != "testTitle" {
		t.Errorf("doc.Title = %q, want testTitle", doc.Title)
	}
}

func TestPostDocument_Errors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		cancelCtx  bool
		wantSubstr string
	}{
		{
			name:       "401 with error body",
			status:     http.StatusUnauthorized,
			body:       `{"error":"unauthorized"}`,
			wantSubstr: "unauthorized",
		},
		{
			name:       "500 without body",
			status:     http.StatusInternalServerError,
			body:       "",
			wantSubstr: "server returned 500",
		},
		{
			name:       "201 with invalid JSON",
			status:     http.StatusCreated,
			body:       `{not json`,
			wantSubstr: "decode response",
		},
		{
			name:       "context canceled",
			cancelCtx:  true,
			wantSubstr: "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					w.Write([]byte(tt.body))
				}
			}))
			defer srv.Close()
			c := newClient(t, srv.URL)
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelCtx {
				cancel()
			} else {
				defer cancel()
			}
			_, err := c.PostDocument(
				ctx,
				"t",
				"fn",
				bytes.NewReader([]byte("x")),
				nil,
			)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("err = %v; want substring %q", err, tt.wantSubstr)
			}
		})
	}
}
