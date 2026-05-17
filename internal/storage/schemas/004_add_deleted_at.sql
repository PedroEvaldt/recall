-- +goose Up
-- Add a nullable timestamp used by queries to hide soft-deleted documents.
ALTER TABLE documents
ADD COLUMN deleted_at TIMESTAMP NULL;

-- +goose Down
-- Remove soft-delete metadata.
ALTER TABLE documents
DROP COLUMN deleted_at;
