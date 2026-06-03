package client_test

import (
	"errors"
	"testing"
	"time"

	"github.com/PedroEvaldt/recall/internal/client"
)

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
