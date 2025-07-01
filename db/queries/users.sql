-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    first_name,
    last_name,
    password_hash,
    role,
    plan
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;
