-- name: GetUserAccess :one
SELECT * FROM user_access
WHERE user_id = $1 LIMIT 1;

-- name: UpsertUserAccess :one
INSERT INTO user_access (
  user_id,
  station_ids,
  variables,
  max_days_history
) VALUES (
  $1, $2, $3, $4
)
ON CONFLICT (user_id) DO UPDATE
SET
  station_ids = EXCLUDED.station_ids,
  variables = EXCLUDED.variables,
  max_days_history = EXCLUDED.max_days_history,
  updated_at = now()
RETURNING *;

-- name: DeleteUserAccess :exec
DELETE FROM user_access WHERE user_id = $1;
