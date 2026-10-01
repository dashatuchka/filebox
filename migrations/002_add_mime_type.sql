-- +goose Up
ALTER TABLE files ADD COLUMN mime_type TEXT;

-- +goose Down
ALTER TABLE files DROP COLUMN mime_type;