-- name: CreateAccessRequest :one
INSERT INTO access_requests (
  user_id,
  station_ids,
  all_stations,
  variables,
  all_variables,
  max_days_history,
  reason
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetAccessRequest :one
SELECT * FROM access_requests
WHERE id = $1 LIMIT 1;

-- name: ListAccessRequestsByUser :many
SELECT * FROM access_requests
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListAccessRequests :many
SELECT * FROM access_requests
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAccessRequestsByStatus :many
SELECT * FROM access_requests
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAccessRequests :one
SELECT count(*) FROM access_requests;

-- name: CountAccessRequestsByStatus :one
SELECT count(*) FROM access_requests
WHERE status = $1;

-- name: UpdateAccessRequestStatus :one
UPDATE access_requests
SET
  status = $2,
  reviewed_by = $3,
  review_note = $4,
  reviewed_at = now(),
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteAccessRequest :exec
DELETE FROM access_requests WHERE id = $1;
