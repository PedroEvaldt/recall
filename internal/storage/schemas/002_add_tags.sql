-- +goose Up
-- Add searchable tags to document metadata.
ALTER TABLE documents
ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';

-- GIN index: makes array membership and overlap operators fast.
CREATE INDEX idx_documents_tags ON documents USING GIN (tags);

-- +goose Down
-- Remove the tag index before dropping the column.
DROP INDEX IF EXISTS idx_documents_tags;

ALTER TABLE documents
DROP COLUMN tags;
