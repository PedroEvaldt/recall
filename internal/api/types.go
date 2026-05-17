package api

import (
	"time"

	"github.com/google/uuid"
)

// DocumentResponse is the public JSON shape returned for document list and
// upload responses.
type DocumentResponse struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	MimeType  string    `json:"mime_type"`
	Filename  string    `json:"filename"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// MetaResponse is the compact JSON shape returned when only document metadata
// is requested.
type MetaResponse struct {
	Title     string    `json:"title"`
	MimeType  string    `json:"mime_type"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}
