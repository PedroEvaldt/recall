package handlers

import (
	"errors"
	"io/fs"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	maxUploadSize = 1000 << 20 // 1 GiB total request limit.
	maxMemorySize = 32 << 20   // 32 MiB multipart memory buffer; the rest spills to disk.
	slugMaxLen    = 80
)

// CreateDocument accepts a multipart upload, stores the file content on disk,
// and persists document metadata in PostgreSQL.
func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxMemorySize) // #nosec G120 -- request body is already limited by MaxBytesReader above.
	if err != nil {
		log.Printf("multipart form error: %v", err)
		respondWithError(w, http.StatusBadRequest, "failed to get multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		respondWithError(w, http.StatusBadRequest, "title is required")
		return
	}
	tags := normalizeTags(r.FormValue("tags"))

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("get file from form: %v", err)
		respondWithError(w, http.StatusBadRequest, "failed to get file from form")
		return
	}
	defer func() { _ = file.Close() }()

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

// ListDocuments returns document metadata matching the optional query and limit.
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	var (
		documents  []database.Document
		err        error
		limit      pgtype.Int4
		limitInt32 int
	)

	searchTerm := r.URL.Query().Get("q")
	limitTerm := r.URL.Query().Get("limit")

	if limitTerm != "" {
		limitInt32, err = strconv.Atoi(limitTerm)
		if err != nil {
			log.Printf("convert limit to int: %v", err)
			respondWithError(w, http.StatusBadRequest, "failed to convert limit to int")
			return
		}
		if limitInt32 < 0 {
			log.Printf("negative limit: %v", limitInt32)
			respondWithError(w, http.StatusBadRequest, "limit must be non-negative")
			return
		}
		if limitInt32 > math.MaxInt32 {
			log.Printf("limit out of range: %v", limitInt32)
			respondWithError(w, http.StatusBadRequest, "limit too large")
			return
		}
		limit = pgtype.Int4{
			Int32: int32(limitInt32), // #nosec G109 -- bounds-checked above (0 <= limitInt32 <= math.MaxInt32)
			Valid: true,
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

// GetDocument streams the stored document content for a document ID.
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
	defer func() { _ = file.Close() }()

	w.Header().Set("Content-Type", document.MimeType)
	http.ServeContent(w, r, document.Filename, document.CreatedAt.Time, file)
}

// GetDocumentMeta returns metadata for a single document ID.
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

var slugTransformer = transform.Chain(
	norm.NFD,
	runes.Remove(runes.In(unicode.Mn)),
	norm.NFC,
)

func slugify(s string) string {
	s, _, _ = transform.String(slugTransformer, s)
	s = strings.ToLower(s)

	var b strings.Builder
	b.Grow(len(s))
	prevDash := true

	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > slugMaxLen {
		out = strings.TrimRight(out[:slugMaxLen], "-")
	}
	return out
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
