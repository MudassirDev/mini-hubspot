-- +goose Up
ALTER TABLE users
ADD COLUMN verification_token TEXT,
ADD COLUMN token_sent_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users
DROP COLUMN verification_token,
DROP COLUMN token_sent_at;
