-- name: CreateMisolStation :one
INSERT INTO misol_station (
  id,
  station_id
) VALUES (
  $1, $2
) RETURNING *;

-- name: GetMisolStation :one
SELECT * FROM misol_station
WHERE id = $1 LIMIT 1;

-- name: DeleteMisolStation :exec
DELETE FROM misol_station WHERE id = $1;
