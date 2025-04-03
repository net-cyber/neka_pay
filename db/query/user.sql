-- name: CreateUser :one
INSERT INTO users (username, hashed_password, full_name, international_phone_number) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

-- name: UpdateUser :one 
UPDATE users SET hashed_password = $2, full_name = $3, international_phone_number = $4 WHERE username = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE username = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY username LIMIT $1 OFFSET $2;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE international_phone_number = $1
LIMIT 1;