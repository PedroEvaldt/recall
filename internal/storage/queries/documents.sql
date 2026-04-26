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
    sqlc.arg (create_title)::text,
    sqlc.arg (create_slug)::text,
    sqlc.arg (create_filename)::text,
    sqlc.arg (create_mime_type)::text,
    sqlc.arg (create_size_bytes)::int,
    sqlc.arg (create_storage_path)::text
  )
RETURNING
  *;

-- name: GetDocument :many
SELECT
  *
FROM
  documents
WHERE
  title ILIKE '%' || sqlc.arg (search_title)::text || '%'
ORDER BY
  created_at DESC;

