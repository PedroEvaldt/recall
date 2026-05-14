package handlers

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxUploadSize = 1000 << 20 // 1Gb — limite total da requisição
	maxMemorySize = 32 << 20   // 32Mb — buffer em memória; resto vai para disco
)

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxMemorySize) // #nosec G120 -- body já limitado por MaxBytesReader acima
	if err != nil {
		log.Printf("multipart form error: %v", err)
		respondWithError(w, http.StatusBadRequest, "failed to get multipart form")
		return
	}

	// Pega campo de texto (chama ParseMultipartForm implicitamente se ainda não parseou)
	title := r.FormValue("title")
	tags := normalizeTags(r.FormValue("tags"))

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
		SizeBytes:   size,
		StoragePath: path,
		Tags:        tags,
	})
	if err != nil {
		log.Printf("create document: %v", err)
		if delErr := h.fileStore.DeleteFile(path); delErr != nil {
			log.Printf("failed to cleanup orphan file: %v", delErr)
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
		Tags:      document.Tags,
		CreatedAt: document.CreatedAt.Time,
	})
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	var (
		documents []database.Document
		err       error
		limit     pgtype.Int4
	)

	searchTerm := r.URL.Query().Get("q")
	limitTerm := r.URL.Query().Get("limit")

	if limitTerm != "" {
		limitInt32, err := strconv.Atoi(limitTerm)
		limit = pgtype.Int4{
			Int32: int32(limitInt32),
			Valid: true,
		}
		if err != nil {
			log.Printf("convert limit to int: %v", err)
			respondWithError(w, http.StatusBadRequest, "failed to convert limit to int")
			return
		}
	}

	if searchTerm == "" {
		documents, err = h.queries.ListAllDocuments(r.Context(), limit)
		if err != nil {
			log.Printf("list all documents: %v", err)
			respondWithError(w, http.StatusInternalServerError, "failed to list all documents")
			return
		}
	} else {
		documents, err = h.queries.ListDocuments(r.Context(), database.ListDocumentsParams{SearchTerm: searchTerm, Limit: limit})
		if err != nil {
			log.Printf("list documents: %v", err)
			respondWithError(w, http.StatusInternalServerError, "failed to list documents")
			return
		}
	}
	response := make([]api.DocumentResponse, 0, len(documents))
	for _, document := range documents {
		if !h.fileStore.Exists(document.StoragePath) {
			log.Printf("list: skipping %s — storage file missing at %s", document.Title, document.StoragePath)
			continue
		}
		id, _ := uuid.FromBytes(document.ID.Bytes[:])
		response = append(response, api.DocumentResponse{
			ID:        id,
			Title:     document.Title,
			MimeType:  document.MimeType,
			Filename:  document.Filename,
			Tags:      document.Tags,
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
		if errors.Is(err, fs.ErrNotExist) {
			log.Printf("get file: storage file missing at %s", document.StoragePath)
			respondWithError(w, http.StatusNotFound, "document content has been removed from storage")
			return
		}
		log.Printf("get file: %v", err)
		respondWithError(w, http.StatusInternalServerError, "failed to get file")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", document.MimeType)
	http.ServeContent(w, r, document.Filename, document.CreatedAt.Time, file)
}

func (h *Handler) GetDocumentMeta(w http.ResponseWriter, r *http.Request) {
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
	respondWithJSON(w, http.StatusOK, api.MetaResponse{
		Title:     document.Title,
		MimeType:  document.MimeType,
		Tags:      document.Tags,
		CreatedAt: document.CreatedAt.Time,
	})
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// produção: usar github.com/gosimple/slug
	return s
}

func normalizeTags(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		tag := strings.TrimSpace(strings.ToLower(part))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		tags = append(tags, tag)
		seen[tag] = struct{}{}

	}
	return tags
}
