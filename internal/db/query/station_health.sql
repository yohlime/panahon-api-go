-- name: CreateStationHealth :one
INSERT INTO observations_stationhealth (
  vb1,
  vb2,
  curr,
  bp1,
  bp2,
  cm,
  ss,
  temp_arq,
  rh_arq,
  fpm,
  error_msg,
  message,
  data_count,
  data_status,
  timestamp,
  minutes_difference,
  station_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetStationHealth :one
SELECT * FROM observations_stationhealth
WHERE station_id = $1 AND id = $2 LIMIT 1;

-- name: ListStationHealths :many
SELECT * FROM observations_stationhealth
WHERE station_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateStationHealth :one
UPDATE observations_stationhealth
SET
  vb1 = COALESCE(sqlc.narg(vb1), vb1),
  vb2 = COALESCE(sqlc.narg(vb2), vb2),
  curr = COALESCE(sqlc.narg(curr), curr),
  bp1 = COALESCE(sqlc.narg(bp1), bp1),
  bp2 = COALESCE(sqlc.narg(bp2), bp2),
  cm = COALESCE(sqlc.narg(cm), cm),
  ss = COALESCE(sqlc.narg(ss), ss),
  temp_arq = COALESCE(sqlc.narg(temp_arq), temp_arq),
  rh_arq = COALESCE(sqlc.narg(rh_arq), rh_arq),
  fpm = COALESCE(sqlc.narg(fpm), fpm),
  error_msg = COALESCE(sqlc.narg(error_msg), error_msg),
  message = COALESCE(sqlc.narg(message), message),
  data_count = COALESCE(sqlc.narg(data_count), data_count),
  data_status = COALESCE(sqlc.narg(data_status), data_status),
  timestamp = COALESCE(sqlc.narg(timestamp), timestamp),
  minutes_difference = COALESCE(sqlc.narg(minutes_difference), minutes_difference),
  updated_at = now()
WHERE station_id = sqlc.arg(station_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteStationHealth :exec
DELETE FROM observations_stationhealth WHERE station_id = $1 AND id = $2;
