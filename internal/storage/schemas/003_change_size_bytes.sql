-- +goose Up
-- Allow file sizes larger than the PostgreSQL INT range.
ALTER TABLE documents
ALTER COLUMN size_bytes TYPE BIGINT;

-- +goose Down
-- Restore the original INT size column type.
ALTER TABLE documents
ALTER COLUMN size_bytes TYPE INT;
