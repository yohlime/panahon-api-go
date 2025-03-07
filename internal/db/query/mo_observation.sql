-- name: CreateStationMOObservation :one
INSERT INTO observations_mo_observation (
  pres,
  rr,
  rh,
  temp,
  td,
  wdir,
  wspd,
  wspdx,
  srad,
  hi,
  wchill,
  timestamp,
  qc_level,
  station_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetStationMOObservation :one
SELECT * FROM observations_mo_observation
WHERE station_id = $1 AND id = $2 LIMIT 1;

-- name: ListStationMOObservations :many
SELECT * FROM observations_mo_observation
WHERE station_id = @station_id
  AND (CASE WHEN @is_start_date::bool THEN timestamp >= @start_date ELSE TRUE END)
  AND (CASE WHEN @is_end_date::bool THEN timestamp <= @end_date ELSE TRUE END)
ORDER BY timestamp DESC
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');

-- name: ListMOObservations :many
SELECT * FROM observations_mo_observation
WHERE station_id = ANY(@station_ids::bigint[])
  AND (CASE WHEN @is_start_date::bool THEN timestamp >= @start_date ELSE TRUE END)
  AND (CASE WHEN @is_end_date::bool THEN timestamp <= @end_date ELSE TRUE END)
ORDER BY timestamp DESC
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');

-- name: CountStationMOObservations :one
SELECT count(*) FROM observations_mo_observation
WHERE station_id = @station_id
  AND (CASE WHEN @is_start_date::bool THEN timestamp >= @start_date ELSE TRUE END)
  AND (CASE WHEN @is_end_date::bool THEN timestamp <= @end_date ELSE TRUE END);

-- name: CountMOObservations :one
SELECT count(*) FROM observations_mo_observation
WHERE station_id = ANY(@station_ids::bigint[])
  AND (CASE WHEN @is_start_date::bool THEN timestamp >= @start_date ELSE TRUE END)
  AND (CASE WHEN @is_end_date::bool THEN timestamp <= @end_date ELSE TRUE END);

-- name: UpdateStationMOObservation :one
UPDATE observations_mo_observation
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
  hi = COALESCE(sqlc.narg(hi), hi),
  wchill = COALESCE(sqlc.narg(wchill), wchill),
  timestamp = COALESCE(sqlc.narg(timestamp), timestamp),
  qc_level = COALESCE(sqlc.narg(qc_level), qc_level),
  updated_at = now()
WHERE station_id = sqlc.arg(station_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteStationMOObservation :exec
DELETE FROM observations_mo_observation WHERE station_id = $1 AND id = $2;
