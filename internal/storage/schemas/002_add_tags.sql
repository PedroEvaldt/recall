-- +goose Up 
ALTER TABLE documents
ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';

-- GIN Indice: turn 'tags @> ARRAY['x'] and 'tags && ARRAY['x'] fast
CREATE INDEX idx_documents_tags ON documents USING GIN (tags);

-- +goose Down
DROP INDEX IF EXISTS idx_documents_tags;

ALTER TABLE documents
DROP COLUMN tags;

