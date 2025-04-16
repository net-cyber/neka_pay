-- name: CreateExternalTransfer :one
INSERT INTO external_transfers (
  from_account_id,
  to_bank_code,
  to_account_number,
  recipient_name,
  amount,
  currency,
  reference,
  description
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetExternalTransfer :one
SELECT * FROM external_transfers WHERE id = $1 LIMIT 1;

-- name: GetExternalTransferByReference :one
SELECT * FROM external_transfers WHERE reference = $1 LIMIT 1;

-- name: ListExternalTransfers :many
SELECT * FROM external_transfers
WHERE from_account_id = $1
ORDER BY id DESC
LIMIT $2
OFFSET $3;

-- name: UpdateExternalTransferStatus :one
UPDATE external_transfers
SET
  status = $2,
  transaction_id = COALESCE($3, transaction_id),
  transaction_fees = COALESCE($4, transaction_fees),
  error_message = COALESCE($5, error_message),
  updated_at = now()
WHERE id = $1
RETURNING *; 