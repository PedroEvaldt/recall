package handlers

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxUploadSize = 50 << 20 // 50 MiB

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		log.Printf("multipart form error: %v", err)
		respondWithError(w, http.StatusBadRequest, "failed to get multipart form")
		return
	}

	// Pega campo de texto (chama ParseMultipartForm implicitamente se ainda não parseou)
	title := r.FormValue("title")

	// Pega arquivo
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("get file from form: %v", err)
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
		log.Printf("save file: %v", err)
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

	id, _ := uuid.FromBytes(document.ID.Bytes[:])
	respondWithJSON(w, http.StatusCreated, api.DocumentResponse{
		ID:        id,
		Title:     document.Title,
		MimeType:  document.MimeType,
		Filename:  document.Filename,
		CreatedAt: document.CreatedAt.Time,
	})
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// produção: usar github.com/gosimple/slug
	return s
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	content := r.URL.Query().Get("q")
	if content == "" {
		respondWithError(w, http.StatusBadRequest, "could not find query")
		return
	}
	documents, err := h.queries.ListDocuments(r.Context(), content)
	if err != nil {
		log.Printf("list documents: %v", err)
		respondWithError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	response := make([]api.DocumentResponse, 0, len(documents))
	for _, document := range documents {
		id, _ := uuid.FromBytes(document.ID.Bytes[:])
		response = append(response, api.DocumentResponse{
			ID:        id,
			Title:     document.Title,
			MimeType:  document.MimeType,
			Filename:  document.Filename,
			CreatedAt: document.CreatedAt.Time,
		})
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	docID := r.PathValue("id")
	docUUID, err := uuid.Parse(docID)
	if err != nil {
		log.Printf("transform id to uuid: %v", err)
		respondWithError(w, http.StatusBadRequest, "could not transform this id to uuid")
		return
	}
	pgID := pgtype.UUID{Bytes: docUUID, Valid: true}
	document, err := h.queries.GetDocument(r.Context(), pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "uuid does not exist")
			return
		}
		log.Printf("get document: %v", err)
		respondWithError(w, http.StatusInternalServerError, "failed to get document")
		return
	}

	file, err := h.fileStore.OpenFile(document.StoragePath)
	if err != nil {
		log.Printf("get file: %v", err)
		respondWithError(w, http.StatusInternalServerError, "failed to get file")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", document.MimeType)
	http.ServeContent(w, r, document.Filename, document.CreatedAt.Time, file)
}
