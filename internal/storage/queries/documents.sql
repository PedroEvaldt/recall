-- name: CreateDocument :one
-- Insert metadata for a file already saved in the filesystem store.
INSERT INTO
  documents (
    title,
    slug,
    filename,
    mime_type,
    size_bytes,
    storage_path,
    tags
  )
VALUES
  (
    sqlc.arg (title)::text,
    sqlc.arg (slug)::text,
    sqlc.arg (filename)::text,
    sqlc.arg (mime_type)::text,
    sqlc.arg (size_bytes)::bigint,
    sqlc.arg (storage_path)::text,
    sqlc.arg (tags)::text[]
  )
RETURNING
  *;

-- name: ListDocuments :many
-- Search active documents by title or tag, newest first.
SELECT
  *
FROM
  documents
WHERE
  (
    (
      title ILIKE '%' || sqlc.arg (search_term)::text || '%'
    )
    OR EXISTS (
      SELECT
        1
      FROM
        unnest(tags) AS t
      WHERE
        t ILIKE '%' || sqlc.arg (search_term)::text || '%'
    )
  )
  AND deleted_at IS NULL
ORDER BY
  created_at DESC
LIMIT
  sqlc.narg ('limit');

-- name: ListAllDocuments :many
-- List every active document, newest first.
SELECT
  *
FROM
  documents
WHERE
  deleted_at IS NULL
ORDER BY
  created_at DESC
LIMIT
  sqlc.narg ('limit');

-- name: GetDocument :one
-- Return metadata required to serve one active document and locate its stored content.
SELECT
  title,
  filename,
  mime_type,
  storage_path,
  size_bytes,
  tags,
  created_at
FROM
  documents
WHERE
  id = sqlc.arg (id)::uuid
  AND deleted_at IS NULL;
