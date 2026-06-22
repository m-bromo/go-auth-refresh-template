-- name: SavePasswordResetToken :exec
INSERT INTO password_reset_tokens (
    id, user_id, token_hash, expires_at
) VALUES (
    $1, $2, $3, $4
);

-- name: GetValidPasswordResetToken :one
SELECT *
FROM password_reset_tokens
WHERE token_hash = $1
    AND used_at IS NULL
    AND expires_at > NOW();

-- name: ConsumePasswordResetToken :one
UPDATE password_reset_tokens
SET used_at = NOW()
WHERE token_hash = $1
    AND used_at IS NULL
    AND expires_at > NOW()
RETURNING *;

-- name: DeletePasswordResetTokensByUserID :exec
DELETE FROM password_reset_tokens
WHERE user_id = $1;

-- name: DeletePasswordResetTokensByID :exec
DELETE FROM password_reset_tokens
WHERE id = $1;