-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    first_name,
    last_name,
    password_hash,
    role,
    plan,
    verification_token,
    token_sent_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: DeleteExpiredUnverifiedUsers :exec
DELETE FROM users
WHERE email_verified = false
  AND token_sent_at < NOW() - INTERVAL '30 days';

-- name: GetUserByVerificationToken :one
SELECT * FROM users
WHERE verification_token = $1;

-- name: VerifyUserEmail :exec
UPDATE users
SET email_verified = true,
    verification_token = NULL,
    token_sent_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: UpgradeUserPlanByEmail :exec
UPDATE users
SET plan = 'pro', updated_at = NOW()
WHERE email = $1;
