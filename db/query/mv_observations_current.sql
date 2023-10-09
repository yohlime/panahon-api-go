-- name: RefreshMVCurrentObservations :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_observations_current;

-- name: ListLatestObservations :many
SELECT
  stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
  sqlc.embed(obs)
FROM observations_station stn 
  JOIN mv_observations_current obs 
  ON stn.id = obs.station_id;


-- name: GetLatestStationObservation :one
SELECT
  stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
  sqlc.embed(obs)
FROM observations_station stn 
  JOIN mv_observations_current obs 
  ON stn.id = obs.station_id
WHERE stn.id = $1 LIMIT 1;