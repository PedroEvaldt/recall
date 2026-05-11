-- +goose Up
ALTER TABLE documents
ADD COLUMN deleted_at TIMESTAMP NULL;

-- +goose Down
ALTER TABLE documents
DROP COLUMN deleted_at;

