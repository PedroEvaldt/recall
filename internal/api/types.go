package api

import (
	"time"

	"github.com/google/uuid"
)

type DocumentResponse struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	MimeType  string    `json:"mime_type"`
	Filename  string    `json:"filename"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

type MetaResponse struct {
	Title     string    `json:"title"`
	MimeType  string    `json:"mime_type"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}
