-- name: createTransfer :one
INSERT INTO transfers (from_account_id, to_account_id, amount) VALUES ($1, $2, $3) RETURNING *;

-- name: getTransfer :one
SELECT * FROM transfers WHERE id = $1 LIMIT 1;

-- name: ListTransfers :many
SELECT * FROM transfers WHERE from_account_id = $1 OR to_account_id = $1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: updateTransfer :one
UPDATE transfers SET amount = $2 WHERE id = $1 RETURNING *;

-- name: deleteTransfer :exec
DELETE FROM transfers WHERE id = $1;