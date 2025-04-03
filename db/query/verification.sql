-- name: CreateOTPVerification :one
INSERT INTO otp_verifications (
    phone_number, 
    otp, 
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetLatestOTPVerification :one
SELECT * FROM otp_verifications
WHERE phone_number = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkOTPAsVerified :one
UPDATE otp_verifications
SET verified = true
WHERE id = $1
RETURNING *;

-- name: UpdateUserPhoneVerified :exec
UPDATE users
SET phone_verified = true
WHERE international_phone_number = $1; 