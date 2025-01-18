-- name: createEntry :one
INSERT INTO entries (account_id, amount) VALUES ($1, $2) RETURNING *;

-- name: getEntry :one
SELECT * FROM entries WHERE id = $1 LIMIT 1;

-- name: ListEntries :many
SELECT * FROM entries WHERE account_id = $1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: updateEntry :one
UPDATE entries SET amount = $2 WHERE id = $1 RETURNING *;

-- name: deleteEntry :exec
DELETE FROM entries WHERE id = $1;