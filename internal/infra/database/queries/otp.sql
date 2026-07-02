-- name: SaveOtpCode :exec
INSERT INTO otp (
    id, identifier, code_hash, attempts, expires_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: InvalidateOtpCodesByIdentifier :exec
DELETE FROM otp
WHERE identifier = $1;

-- name: ConsumeOtpCode :one
DELETE FROM otp
WHERE code_hash = $1
    AND identifier = $2
    AND expires_at > NOW()
RETURNING *;

-- name: ConsumeOtpCodeByChallengeID :one
DELETE FROM otp
WHERE id = $1
    AND code_hash = $2
    AND expires_at > NOW()
    AND attempts < sqlc.arg(max_attempts)
RETURNING *;

-- name: IncreaseOtpAttempts :exec
UPDATE otp
SET attempts = attempts + 1
WHERE id = $1
    AND expires_at > NOW()
    AND attempts < sqlc.arg(max_attempts);
