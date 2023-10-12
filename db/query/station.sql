-- name: CreateStation :one
INSERT INTO observations_station (
  name,
  lat,
  lon,
  elevation,
  date_installed,
  mo_station_id,
  sms_system_type,
  mobile_number,
  station_type,
  station_type2,
  station_url,
  status,
  logger_version,
  priority_level,
  provider_id,
  province,
  region,
  address,
  geom
) VALUES (
  $1, @lat, @lon, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, 
  CASE
    WHEN @lon::real IS NOT NULL AND @lat::real IS NOT NULL THEN ST_Point(@lon::real, @lat::real, 4326)
    ELSE ST_GeomFromEWKT('POINT EMPTY')
  END
) RETURNING *;

-- name: GetStation :one
SELECT * FROM observations_station
WHERE id = $1 LIMIT 1;

-- name: GetStationByMobileNumber :one
SELECT * FROM observations_station
WHERE mobile_number = $1 LIMIT 1;

-- name: ListStations :many
SELECT * FROM observations_station
ORDER BY id
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');

-- name: ListStationsWithinRadius :many
SELECT * FROM observations_station
WHERE ST_DWithin(geom, ST_Point(@cx::real, @cy::real, 4326), @r::real)
ORDER BY id
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');

-- name: ListStationsWithinBBox :many
SELECT * FROM observations_station
WHERE geom && ST_MakeEnvelope(@xmin::real, @ymin::real, @xmax::real, @ymax::real, 4326)
ORDER BY id
LIMIT sqlc.narg('limit')
OFFSET sqlc.arg('offset');

-- name: CountStations :one
SELECT count(*) FROM observations_station;

-- name: CountStationsWithinRadius :one
SELECT count(*) FROM observations_station
WHERE ST_DWithin(geom, ST_Point(@cx::real, @cy::real, 4326), @r::real);

-- name: CountStationsWithinBBox :one
SELECT count(*) FROM observations_station
WHERE geom && ST_MakeEnvelope(@xmin::real, @ymin::real, @xmax::real, @ymax::real, 4326);

-- name: UpdateStation :one
UPDATE observations_station
SET
  name = COALESCE(sqlc.narg(name), name),
  lat = COALESCE(sqlc.narg(lat), lat),
  lon = COALESCE(sqlc.narg(lon), lon),
  elevation = COALESCE(sqlc.narg(elevation), elevation),
  date_installed = COALESCE(sqlc.narg(date_installed), date_installed),
  mo_station_id = COALESCE(sqlc.narg(mo_station_id), mo_station_id),
  sms_system_type = COALESCE(sqlc.narg(sms_system_type), sms_system_type),
  mobile_number = COALESCE(sqlc.narg(mobile_number), mobile_number),
  station_type = COALESCE(sqlc.narg(station_type), station_type),
  station_type2 = COALESCE(sqlc.narg(station_type2), station_type2),
  station_url = COALESCE(sqlc.narg(station_url), station_url),
  status = COALESCE(sqlc.narg(status), status),
  logger_version = COALESCE(sqlc.narg(logger_version), logger_version),
  priority_level = COALESCE(sqlc.narg(priority_level), priority_level),
  provider_id = COALESCE(sqlc.narg(provider_id), provider_id),
  province = COALESCE(sqlc.narg(province), province),
  region = COALESCE(sqlc.narg(region), region),
  address = COALESCE(sqlc.narg(address), address),
  geom = COALESCE(ST_POINT(sqlc.narg(lon), sqlc.narg(lat), 4326), geom),
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteStation :exec
DELETE FROM observations_station WHERE id = $1;
