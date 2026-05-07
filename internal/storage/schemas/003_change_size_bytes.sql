-- +goose Up
ALTER TABLE documents
ALTER COLUMN size_bytes TYPE BIGINT;

-- +goose Down
ALTER TABLE documents
ALTER COLUMN size_bytes TYPE INT;

