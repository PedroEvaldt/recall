package handlers

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/google/uuid"
)

const maxUploadSize = 50 << 20 // 50 MiB

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failed to get multipart form")
		return
	}

	// Pega campo de texto (chama ParseMultipartForm implicitamente se ainda não parseou)
	title := r.FormValue("title")

	// Pega arquivo
	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failed to get file from form")
		return
	}
	defer file.Close()

	// Getting from header
	filename := header.Filename
	extension := filepath.Ext(header.Filename)
	mimeType := header.Header.Get("Content-Type")
	uuidInt := uuid.New()
	path, size, err := h.fileStore.SaveFile(uuidInt, extension, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	docSlug := slugify(title)

	document, err := h.queries.CreateDocument(r.Context(), database.CreateDocumentParams{
		Title:       title,
		Slug:        docSlug,
		Filename:    filename,
		MimeType:    mimeType,
		SizeBytes:   int32(size),
		StoragePath: path,
	})
	if err != nil {
		if delErr := h.fileStore.DeleteFile(path); delErr != nil {
			log.Printf("failed to cleanup orphan file %s: %v", path, delErr)
		}
		respondWithError(w, http.StatusInternalServerError, "failed to save document")
		return
	}
	respondWithJSON(w, http.StatusCreated, document)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// produção: usar github.com/gosimple/slug
	return s
}
