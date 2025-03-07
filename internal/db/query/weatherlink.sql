-- name: CreateWeatherlinkStation :one
INSERT INTO weatherlink (
  station_id,
  uuid,
  api_key,
  api_secret
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: ListWeatherlinkStations :many
SELECT * FROM weatherlink
WHERE
  uuid IS NOT NULL OR (api_key IS NOT NULL AND api_secret IS NOT NULL)
ORDER BY id
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');
