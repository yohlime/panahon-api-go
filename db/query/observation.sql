-- name: CreateStationObservation :one
INSERT INTO observations_observation (
  pres,
  rr,
  rh,
  temp,
  td,
  wdir,
  wspd,
  wspdx,
  srad,
  mslp,
  hi,
  wchill,
  timestamp,
  qc_level,
  station_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING *;

-- name: GetStationObservation :one
SELECT * FROM observations_observation
WHERE station_id = $1 AND id = $2 LIMIT 1;

-- name: ListStationObservations :many
SELECT * FROM observations_observation
WHERE station_id = @station_id
  AND (CASE WHEN @is_start_date::bool THEN timestamp >= @start_date ELSE TRUE END)
  AND (CASE WHEN @is_end_date::bool THEN timestamp <= @end_date ELSE TRUE END)
ORDER BY timestamp DESC
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');

-- name: CountStationObservations :one
SELECT count(*) FROM observations_observation
WHERE station_id = @station_id
  AND (CASE WHEN @is_start_date::bool THEN timestamp >= @start_date ELSE TRUE END)
  AND (CASE WHEN @is_end_date::bool THEN timestamp <= @end_date ELSE TRUE END);

-- name: UpdateStationObservation :one
UPDATE observations_observation
SET
  pres = COALESCE(sqlc.narg(pres), pres),
  rr = COALESCE(sqlc.narg(rr), rr),
  rh = COALESCE(sqlc.narg(rh), rh),
  temp = COALESCE(sqlc.narg(temp), temp),
  td = COALESCE(sqlc.narg(td), td),
  wdir = COALESCE(sqlc.narg(wdir), wdir),
  wspd = COALESCE(sqlc.narg(wspd), wspd),
  wspdx = COALESCE(sqlc.narg(wspdx), wspdx),
  srad = COALESCE(sqlc.narg(srad), srad),
  mslp = COALESCE(sqlc.narg(mslp), mslp),
  hi = COALESCE(sqlc.narg(hi), hi),
  wchill = COALESCE(sqlc.narg(wchill), wchill),
  timestamp = COALESCE(sqlc.narg(timestamp), timestamp),
  qc_level = COALESCE(sqlc.narg(qc_level), qc_level),
  updated_at = now()
WHERE station_id = sqlc.arg(station_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteStationObservation :exec
DELETE FROM observations_observation WHERE station_id = $1 AND id = $2;
