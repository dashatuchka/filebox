-- +goose
ALTER TABLE files ADD COLUMN mime_type TEXT;

-- +goose
ALTER TABLE files DROP COLUMN mime_type;