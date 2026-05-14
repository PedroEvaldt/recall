-- name: CreateDocument :one
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
  created_at DESC;

-- name: ListAllDocuments :many
SELECT
  *
FROM
  documents
WHERE
  deleted_at IS NULL
ORDER BY
  created_at DESC;

-- name: GetDocument :one
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

