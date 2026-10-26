-- name: CreateUserToken :one
INSERT INTO user_tokens (
  user_id,
  purpose,
  token_hash,
  expires_at,
  created_by
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: InvalidateUserTokens :exec
UPDATE user_tokens
SET used_at = now()
WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL;

-- name: GetUserTokenByHash :one
SELECT * FROM user_tokens
WHERE token_hash = $1 AND purpose = $2
LIMIT 1;

-- name: ConsumeUserToken :one
UPDATE user_tokens
SET used_at = now()
WHERE token_hash = $1
  AND purpose = $2
  AND used_at IS NULL
  AND expires_at > now()
RETURNING *;
