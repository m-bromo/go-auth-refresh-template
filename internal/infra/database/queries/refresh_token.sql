-- name: SaveRefreshToken :exec
INSERT INTO refresh_tokens (
    id, user_id, created_at, expires_at
) VALUES (
    $1, $2, $3, $4
);

-- name: GetRefreshTokenByID :one
SELECT *
FROM refresh_tokens
WHERE id = $1
    AND expires_at > NOW();

-- name: ConsumeRefreshToken :one
UPDATE refresh_tokens
SET expires_at = NOW()
WHERE id = $1
    AND expires_at > NOW()
RETURNING *;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE id = $1;

-- name: DeleteRefreshTokensByUserID :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;