-- name: CreateTransfer :one
INSERT INTO transfer (from_account_id, to_account_id, amount) VALUES ($1, $2, $3) RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfer WHERE id = $1 LIMIT 1 FOR UPDATE;

-- name: ListTransfers :many
SELECT * FROM transfer WHERE from_account_id = $1 OR to_account_id = $1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: UpdateTransfer :one
UPDATE transfer SET amount = $2 WHERE id = $1 RETURNING *;

-- name: DeleteTransfer :exec
DELETE FROM transfer WHERE id = $1;
