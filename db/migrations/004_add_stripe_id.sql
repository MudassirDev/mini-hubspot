-- +goose Up
ALTER TABLE users ADD COLUMN stripe_customer_id TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN stripe_customer_id;
