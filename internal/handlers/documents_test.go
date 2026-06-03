//go:build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PedroEvaldt/recall/internal/api"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestCreateDocuments(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		file   bool
		absent bool
		err    string
		code   int
	}{
		{
			name:   "Complete Multipart",
			title:  "Go structs",
			file:   true,
			absent: false,
			code:   http.StatusCreated,
			err:    "",
		},
		{
			name:   "Multipart without file field",
			title:  "Go structs",
			absent: false,
			file:   false,
			code:   http.StatusBadRequest,
			err:    "failed to get file from form",
		},
		{
			name:   "No multipart",
			title:  "Go structs",
			absent: true,
			file:   false,
			code:   http.StatusBadRequest,
			err:    "failed to get multipart form",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var req *http.Request
			ctx := context.Background()
			h, ctr := newTestServer(t)

			t.Cleanup(func() {
				if err := ctr.Restore(ctx); err != nil {
					t.Fatalf("could not restore container to previous snapshot: %v", err)
				}
			})

			if tt.absent {
				req = httptest.NewRequest("POST", "/document", nil)
			} else {
				req = httptest.NewRequest("POST", "/document", &buf)
				multWriter := multipart.NewWriter(&buf)
				if err := multWriter.WriteField("title", tt.title); err != nil {
					t.Fatalf("could not write title field in multipart writer: %v", err)
				}
				if tt.file {
					fileWriter, err := multWriter.CreateFormFile("file", "go-structs.md")
					if err != nil {
						t.Fatalf("could not create writer from multipart writer: %v", err)
					}
					if _, err := fmt.Fprint(fileWriter, "#Go struct file body"); err != nil {
						t.Fatalf("could not write body into file writer: %v", err)
					}
				}
				req.Header.Set("Content-Type", multWriter.FormDataContentType())
				if err := multWriter.Close(); err != nil {
					t.Fatalf("could not close multipart writer: %v", err)
				}
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
