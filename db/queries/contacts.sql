-- name: CreateContact :one
INSERT INTO contacts (
    user_id, name, email, phone, company, position, notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CountContactsByUser :one
SELECT COUNT(*) FROM contacts
WHERE user_id = $1;

-- name: GetContactsByUser :many
SELECT * FROM contacts
WHERE user_id = $1;

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

-- name: GetContactsPaginated :many
SELECT *
FROM contacts
WHERE user_id = sqlc.arg('user_id')
  AND id > sqlc.arg('after')
  AND (
    sqlc.arg('search')::text IS NULL OR
    name ILIKE '%' || sqlc.arg('search') || '%' OR
    email ILIKE '%' || sqlc.arg('search') || '%' OR
    phone ILIKE '%' || sqlc.arg('search') || '%'
  )
  AND (
    NOT sqlc.arg('require_non_empty_phone')::bool OR (phone IS NOT NULL AND phone <> '')
  )
  AND (
    NOT sqlc.arg('require_non_empty_company')::bool OR (company IS NOT NULL AND company <> '')
  )
  AND (
    NOT sqlc.arg('require_non_empty_position')::bool OR (position IS NOT NULL AND position <> '')
  )
  AND (
    NOT sqlc.arg('require_non_empty_email')::bool OR (email IS NOT NULL AND email <> '')
  )
ORDER BY id
LIMIT sqlc.arg('limit');
