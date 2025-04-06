-- name: CreateFinancialInstitution :one
INSERT INTO financial_institutions (
  name,
  type,
  logo_url,
  code,
  active
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetFinancialInstitution :one
SELECT * FROM financial_institutions
WHERE id = $1 LIMIT 1;

-- name: GetFinancialInstitutionByCode :one
SELECT * FROM financial_institutions
WHERE code = $1 LIMIT 1;

-- name: ListFinancialInstitutions :many
SELECT * FROM financial_institutions
WHERE 
  (SQLC.NARG(type)::financial_institution_type IS NULL OR type = SQLC.NARG(type)::financial_institution_type)
  AND active = COALESCE(SQLC.NARG(active), active)
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: UpdateFinancialInstitution :one
UPDATE financial_institutions
SET 
  name = COALESCE(SQLC.NARG(name), name),
  logo_url = COALESCE(SQLC.NARG(logo_url), logo_url),
  active = COALESCE(SQLC.NARG(active), active)
WHERE id = $1
RETURNING *;

-- name: DeleteFinancialInstitution :exec
DELETE FROM financial_institutions
WHERE id = $1; 