-- name: CreateSimAccessToken :one
INSERT INTO sim_access_tokens (
  access_token,
  type,
  mobile_number
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetSimAccessToken :one
SELECT * FROM sim_access_tokens
WHERE access_token = $1 LIMIT 1;
