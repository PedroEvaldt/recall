//go:build integration

package handlers_test

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/PedroEvaldt/recall/internal/handlers"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestCreateDocuments(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		hasFile  bool
		absent   bool
		err      string
		filename string
		code     int
	}{
		{
			name:     "Complete Multipart",
			title:    "Go structs",
			hasFile:  true,
			absent:   false,
			code:     http.StatusCreated,
			filename: "go-struct.md",
			err:      "",
		},
		{
			name:     "Multipart without file field",
			title:    "Go structs",
			absent:   false,
			hasFile:  false,
			filename: "go-struct.md",
			code:     http.StatusBadRequest,
			err:      "failed to get file from form",
		},
		{
			name:     "No multipart",
			title:    "Go structs",
			absent:   true,
			hasFile:  false,
			filename: "go-struct.md",
			code:     http.StatusBadRequest,
			err:      "failed to get multipart form",
		},
		{
			name:     "No title",
			title:    "",
			absent:   false,
			hasFile:  true,
			filename: "go-struct.md",
			code:     http.StatusBadRequest,
			err:      "title is required",
		},
		{
			name:     "No extension",
			title:    "Go structs",
			absent:   false,
			hasFile:  true,
			filename: "go-struct_FOO",
			code:     http.StatusInternalServerError,
			err:      "failed to save file",
		},
	}

	h, reset, _ := newTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			t.Cleanup(reset)
			if tt.absent {
				req = httptest.NewRequest("POST", "/documents", nil)
			} else {
				req = newMultipartReq(t, tt.hasFile, tt.title, tt.filename)
			}

			rec := httptest.NewRecorder()
			h.CreateDocument(rec, req)

			if contentHeader := rec.Header().Get("Content-Type"); contentHeader != "application/json" {
				t.Errorf("expected application/json; got %s", contentHeader)
			}

			if rec.Code != tt.code {
				t.Errorf("expected %s; got %s", http.StatusText(tt.code), http.StatusText(rec.Code))
			}

			resp, err := io.ReadAll(rec.Body)
			if err != nil {
				t.Fatalf("could not read request body: %v", err)
			}
			if tt.err != "" {
				errResp := struct {
					Error string `json:"error"`
				}{}
				err = json.Unmarshal(resp, &errResp)
				if err != nil {
					t.Fatalf("could not unmarshal resp to error body: %v", err)
				}
				if tt.err != errResp.Error {
					t.Errorf("expected %s; got %s", tt.err, errResp.Error)
				}
			} else {
				fileResp := api.DocumentResponse{}
				err = json.Unmarshal(resp, &fileResp)
				if err != nil {
					t.Fatalf("could not unmarshal resp to file body: %v", err)
				}

				if fileResp.Title != tt.title {
					t.Errorf("expected %s; got %s", tt.title, fileResp.Title)
				}
			}
		})
	}
}

func TestCreateDocumentCleanOrphanOnDBFailure(t *testing.T) {
	h, _, dir := newTestServer(t)

	rec1 := httptest.NewRecorder()
	h.CreateDocument(rec1, newMultipartReq(t, true, "dup", "test.md"))
	if rec1.Code != http.StatusCreated {
		t.Fatalf("setup upload failed: %d %s", rec1.Code, rec1.Body.String())
	}

	filesBefore := countFiles(t, dir)
	if filesBefore != 1 {
		t.Fatalf("expected 1 file after first upload, got %d", filesBefore)
	}

	rec2 := httptest.NewRecorder()
	h.CreateDocument(rec2, newMultipartReq(t, true, "dup", "test.md"))

	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500; got %v", rec2.Code)
	}

	filesAfter := countFiles(t, dir)
	if filesAfter != filesBefore {
		t.Errorf("orphan leaked: %d files before, %d files after", filesBefore, filesAfter)
	}
}

func TestListDocuments(t *testing.T) {
	tests := []struct {
		name       string
		err        string
		query      string
		limit      string
		code       int
		wantedDocs int
		wantErr    bool
		titles     []string
		orphans    []string
	}{
		{
			name:       "Empty query",
			err:        "",
			query:      "",
			limit:      "",
			wantedDocs: 1,
			code:       http.StatusOK,
			wantErr:    false,
			titles:     []string{"go-struct"},
		},
		{
			name:       "Normal limit",
			err:        "",
			query:      "",
			limit:      "2",
			wantedDocs: 2,
			code:       http.StatusOK,
			wantErr:    false,
			titles:     []string{"go-struct", "sql", "linux"},
		},
		{
			name:       "Normal query",
			err:        "",
			query:      "SQL",
			limit:      "",
			wantedDocs: 1,
			code:       http.StatusOK,
			wantErr:    false,
			titles:     []string{"go-struct", "sql", "linux"},
		},
		{
			name:       "Non existent query",
			err:        "",
			query:      "dontexist",
			limit:      "",
			wantedDocs: 0,
			code:       http.StatusOK,
			wantErr:    false,
			titles:     []string{"go-struct"},
		},
		{
			name:    "String limit",
			err:     "failed to convert limit to int",
			query:   "",
			limit:   "abcd",
			code:    http.StatusBadRequest,
			wantErr: true,
		},
		{
			name:    "Negative limit",
			err:     "limit must be non-negative",
			query:   "",
			limit:   "-5",
			code:    http.StatusBadRequest,
			wantErr: true,
		},
		{
			name:       "Orphan file",
			err:        "",
			query:      "",
			limit:      "",
			wantedDocs: 1,
			code:       http.StatusOK,
			wantErr:    false,
			titles:     []string{"ghost", "real"},
			orphans:    []string{"ghost"},
		},
	}

	h, reset, dir := newTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(reset)

			var remainingFiles []string
			for _, title := range tt.titles {
				if !slices.Contains(tt.orphans, title) {
					remainingFiles = append(remainingFiles, title)
					continue
				}
				seedDocument(t, h, title)
				removeAllFiles(t, dir)
			}
			for _, remainingFile := range remainingFiles {
				seedDocument(t, h, remainingFile)
			}
			rec := httptest.NewRecorder()
			v := url.Values{}
			v.Set("q", tt.query)
			v.Set("limit", tt.limit)
			req := httptest.NewRequest("GET", "/documents?"+v.Encode(), nil)

			h.ListDocuments(rec, req)

			if rec.Code != tt.code {
				t.Errorf("expected code %v; got %v", tt.code, rec.Code)
			}
			if header := rec.Header().Get("Content-Type"); header != "application/json" {
				t.Errorf("expected application/json; got %v", header)
			}
			if !tt.wantErr {
				var docs []api.DocumentResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &docs); err != nil {
					t.Fatalf("unmarshaling: %v", err)
				}
				if len(docs) != tt.wantedDocs {
					t.Errorf("expected %v docs; got %v", tt.wantedDocs, len(docs))
				}
			} else {
				type errResp struct {
					Error string `json:"error"`
				}
				var errBody errResp
				if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
					t.Fatalf("unmarshaling: %v", err)
				}
				if errBody.Error != tt.err {
					t.Errorf("expected error %v; got %v", tt.err, errBody.Error)
				}
			}
		})
	}
}

func TestGetDocument(t *testing.T) {
	tests := []struct {
		name    string
		err     string
		code    int
		wantErr bool
	}{
		{
			name:    "Invalid id",
			err:     "could not transform this id to uuid",
			code:    http.StatusBadRequest,
			wantErr: true,
		},
		{
			name:    "Non existent id",
			err:     "uuid does not exist",
			code:    http.StatusNotFound,
			wantErr: true,
		},
		{
			name:    "Valid id",
			err:     "",
			code:    http.StatusOK,
			wantErr: false,
		},
		{
			name:    "Non existent file",
			err:     "document content has been removed from storage",
			code:    http.StatusNotFound,
			wantErr: true,
		},
	}
	h, reset, dir := newTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(reset)

			doc := seedDocument(t, h, "go struct")

			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/documents/test/content", nil)

			if tt.wantErr {
				switch tt.name {
				case "Invalid id":
					req.SetPathValue("id", "abcd")
				case "Non existent id":
					req.SetPathValue("id", uuid.New().String())
				case "Non existent file":
					req.SetPathValue("id", doc.ID.String())
					removeAllFiles(t, dir)
				}
				h.GetDocument(rec, req)
				if rec.Code != tt.code {
					t.Errorf("expected code %v; got %v", tt.code, rec.Code)
				}
				var body struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshaling: %v", err)
				}
				if body.Error != tt.err {
					t.Errorf("expected error %v; got %v", tt.err, body.Error)
				}

			} else {
				req.SetPathValue("id", doc.ID.String())
				h.GetDocument(rec, req)
				if rec.Code != tt.code {
					t.Errorf("expected code %v; got %v", tt.code, rec.Code)
				}
				const expectedBody = "#Go struct file body"
				if got := rec.Body.String(); got != expectedBody {
					t.Errorf("expected body %q; got %q", expectedBody, got)
				}
				if header := rec.Header().Get("Content-Type"); header != doc.MimeType {
					t.Errorf("expected %s; got %s", doc.MimeType, header)
				}
			}
		})
	}
}

func TestGetDocumentRangeRequest(t *testing.T) {
	h, _, _ := newTestServer(t)
	doc := seedDocument(t, h, "go struct")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/documents/test/content", nil)
	req.SetPathValue("id", doc.ID.String())
	req.Header.Set("Range", "bytes=0-3")

	h.GetDocument(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Errorf("expected %v; got %v", http.StatusPartialContent, rec.Code)
	}
	const expected = "#Go "
	if got := rec.Body.String(); got != expected {
		t.Errorf("expected body %q; got %q", expected, got)
	}
}

func TestGetDocumentMeta(t *testing.T) {
	tests := []struct {
		name    string
		err     string
		code    int
		wantErr bool
	}{
		{
			name:    "Invalid id",
			err:     "could not transform this id to uuid",
			code:    http.StatusBadRequest,
			wantErr: true,
		},
		{
			name:    "Non existent id",
			err:     "uuid does not exist",
			code:    http.StatusNotFound,
			wantErr: true,
		},
		{
			name:    "Valid id",
			err:     "",
			code:    http.StatusOK,
			wantErr: false,
		},
	}
	h, reset, _ := newTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(reset)

			doc := seedDocument(t, h, "go struct")

			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/documents/test", nil)

			if tt.wantErr {
				switch tt.name {
				case "Invalid id":
					req.SetPathValue("id", "abcd")
				case "Non existent id":
					req.SetPathValue("id", uuid.New().String())
				}
				h.GetDocumentMeta(rec, req)
				if rec.Code != tt.code {
					t.Errorf("expected code %v; got %v", tt.code, rec.Code)
				}
				var body struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshaling: %v", err)
				}
				if body.Error != tt.err {
					t.Errorf("expected error %v; got %v", tt.err, body.Error)
				}
			} else {
				req.SetPathValue("id", doc.ID.String())
				h.GetDocumentMeta(rec, req)
				if rec.Code != tt.code {
					t.Errorf("expected code %v; got %v", tt.code, rec.Code)
				}
				var meta api.MetaResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
					t.Fatalf("unmarshaling: %v", err)
				}
				if meta.Title != doc.Title {
					t.Errorf("expected title %q; got %q", doc.Title, meta.Title)
				}
				if meta.MimeType != doc.MimeType {
					t.Errorf("expected mime %q; got %q", doc.MimeType, meta.MimeType)
				}
				if !slices.Equal(meta.Tags, doc.Tags) {
					t.Errorf("expected tags %v; got %v", doc.Tags, meta.Tags)
				}
			}
		})
	}
}

func seedDocument(t *testing.T, h *handlers.Handler, title string) api.DocumentResponse {
	t.Helper()
	req := newMultipartReq(t, true, title, title+".md")
	rec := httptest.NewRecorder()
	h.CreateDocument(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %v; got %v", http.StatusCreated, rec.Code)
	}
	var resp api.DocumentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	return resp
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	var n int
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return n
}

func removeAllFiles(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			err := os.Remove(p)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}
