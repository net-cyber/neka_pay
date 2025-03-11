-- name: CreateEntry :one
INSERT INTO entry (account_id, amount) VALUES ($1, $2) RETURNING *;

-- name: GetEntry :one
SELECT * FROM entry WHERE id = $1 LIMIT 1;

-- name: ListEntries :many
SELECT * FROM entry WHERE account_id = $1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: UpdateEntry :one
UPDATE entry SET amount = $2 WHERE id = $1 RETURNING *;

-- name: DeleteEntry :exec
DELETE FROM entry WHERE id = $1;
