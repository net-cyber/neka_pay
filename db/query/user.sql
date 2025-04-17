-- name: CreateUser :one
INSERT INTO users (username, hashed_password, full_name, international_phone_number, token, avatar, online, fcmtoken) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;
-- name: UpdateUserOnline :exec
UPDATE users SET online = $2 WHERE username = $1;
-- name: GetUser :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

-- name: UpdateUser :one 
UPDATE users SET hashed_password = $2, full_name = $3, international_phone_number = $4, token = $5, avatar = $6 WHERE username = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE username = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY username LIMIT $1 OFFSET $2;

-- name: ListUsersOthers :many
SELECT * FROM users WHERE username != $1 ORDER BY username LIMIT $2 OFFSET $3;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE international_phone_number = $1
LIMIT 1;

-- name: UpdateUserFCMToken :exec
UPDATE users SET fcmtoken = $2 WHERE username = $1;

-- name: GetUserByToken :one
SELECT * FROM users
WHERE token = $1
LIMIT 1;