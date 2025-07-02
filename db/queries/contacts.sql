-- name: CreateContact :one
INSERT INTO contacts (
    user_id, name, email, phone, company, position, notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetContactsByUser :many
SELECT * FROM contacts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetContactByID :one
SELECT * FROM contacts
WHERE id = $1 AND user_id = $2;

-- name: UpdateContact :one
UPDATE contacts
SET name = $3,
    email = $4,
    phone = $5,
    company = $6,
    position = $7,
    notes = $8,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteContact :exec
DELETE FROM contacts
WHERE id = $1 AND user_id = $2;

