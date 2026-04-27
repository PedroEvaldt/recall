-- name: CreateDocument :one 
INSERT INTO
  documents (
    title,
    slug,
    filename,
    mime_type,
    size_bytes,
    storage_path
  )
VALUES
  (
    sqlc.arg (title)::text,
    sqlc.arg (slug)::text,
    sqlc.arg (filename)::text,
    sqlc.arg (mime_type)::text,
    sqlc.arg (size_bytes)::int,
    sqlc.arg (storage_path)::text
  )
RETURNING
  *;

-- name: ListDocuments :many
SELECT
  *
FROM
  documents
WHERE
  title ILIKE '%' || sqlc.arg (search_title)::text || '%'
ORDER BY
  created_at DESC;

