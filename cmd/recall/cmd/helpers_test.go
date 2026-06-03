package cmd

import (
	"testing"

	"github.com/PedroEvaldt/recall/internal/api"
)

func TestIsMarkdownDocument(t *testing.T) {
	tests := []struct {
		name string
		doc  api.DocumentResponse
		want bool
	}{
		{
			name: "Mime text/markdown",
			doc:  api.DocumentResponse{MimeType: "text/markdown"},
			want: true,
		},
		{
			name: "Mime text/x-markdown",
			doc:  api.DocumentResponse{MimeType: "text/x-markdown"},
			want: true,
		},
		{
			name: "Right extension",
			doc:  api.DocumentResponse{Filename: "foo.md"},
			want: true,
		},
		{
			name: "Ext .markdown lowercase",
			doc:  api.DocumentResponse{Filename: "notes.markdown"},
			want: true,
		},
		{
			name: "Uppercase extension",
			doc:  api.DocumentResponse{Filename: "foo.MARKDOWN"},
			want: true,
		},
		{
			name: "Mime uppercase",
			doc:  api.DocumentResponse{MimeType: "TEXT/MARKDOWN"},
			want: true,
		},
		{
			name: "Mime with charset",
			doc:  api.DocumentResponse{MimeType: "text/markdown; charset=utf-8"},
			want: true,
		},
		{
			name: "Wrong mime and filename",
			doc:  api.DocumentResponse{MimeType: "application/pdf", Filename: "foo.pdf"},
			want: false,
		},
		{
			name: "Wrong final extension",
			doc:  api.DocumentResponse{Filename: "README.md.bak"},
			want: false,
		},
		{
			name: "All empty",
			doc:  api.DocumentResponse{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMarkdownDocument(tt.doc)
			if got != tt.want {
				t.Errorf("expected %v; got %v", tt.want, got)
			}
		})
	}
}
