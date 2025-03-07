-- name: ListLatestObservations :many
WITH RankedRows AS (
  SELECT
    stn.id, stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
    obs.rain, obs."temp", obs.rh,
	obs.wdir, obs.wspd, obs.srad, obs.mslp,
	obs.tn, obs.tx, obs.gust, obs.rain_accum,
	obs.tn_timestamp, obs.tx_timestamp, obs.gust_timestamp, obs."timestamp",
    ROW_NUMBER() OVER (PARTITION BY stn.id ORDER BY obs.timestamp DESC) AS rn
  FROM observations_station stn 
    JOIN observations_current obs 
    ON stn.id = obs.station_id
  WHERE obs.timestamp > NOW() - INTERVAL '1 hour'
)
SELECT *
FROM RankedRows
WHERE rn = 1;


-- name: GetLatestStationObservation :one
SELECT
  stn.id, stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
  sqlc.embed(obs)
FROM observations_station stn 
  JOIN observations_current obs 
  ON stn.id = obs.station_id
WHERE stn.id = $1
ORDER BY obs.timestamp DESC
LIMIT 1;

-- name: GetNearestLatestStationObservation :one
WITH NearestStation AS (
  SELECT
	id, name, lat, lon, elevation, address
  FROM observations_station
  ORDER BY geom <-> ST_Point(@lon::real, @lat::real, 4326)
  LIMIT 1
)
SELECT
  stn.id, stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
  sqlc.embed(obs)
FROM NearestStation stn 
  JOIN observations_current obs 
  ON stn.id = obs.station_id
ORDER BY obs.timestamp DESC
LIMIT 1;

-- name: InsertCurrentObservations :many
INSERT INTO observations_current (
  station_id,
	rain, "temp", rh,
	wdir, wspd, srad, mslp,
	tn, tx, gust, rain_accum,
	tn_timestamp, tx_timestamp, gust_timestamp, "timestamp")
SELECT DISTINCT(station_id),
	LAST_VALUE(rr / 6) OVER wdw::real AS rain,
	LAST_VALUE("temp") OVER wdw::real AS "temp",
	LAST_VALUE(rh) OVER wdw::real AS rh,
	LAST_VALUE(wdir) OVER wdw::real AS wdir,
	LAST_VALUE(wspd) OVER wdw::real AS wspd,
	LAST_VALUE(srad) OVER wdw::real AS srad,
	LAST_VALUE(mslp) OVER wdw::real AS mslp,
	FIRST_VALUE("temp") OVER t_wdw::real AS tn,
	LAST_VALUE("temp") OVER t_wdw::real AS tx,
	LAST_VALUE("wspdx") OVER w_wdw::real AS gust,
	SUM(rr / 6) OVER t_wdw::real AS rain_accum,
	FIRST_VALUE("timestamp") OVER t_wdw::TIMESTAMPTZ AS tn_timestamp,
	LAST_VALUE("timestamp") OVER t_wdw::TIMESTAMPTZ AS tx_timestamp,
	LAST_VALUE("timestamp") OVER w_wdw::TIMESTAMPTZ AS gust_timestamp,
	LAST_VALUE("timestamp") OVER wdw::TIMESTAMPTZ AS "timestamp"
FROM observations_observation
WHERE "timestamp" BETWEEN CURRENT_DATE AND CURRENT_TIMESTAMP
WINDOW
  wdw AS (PARTITION BY station_id ORDER BY "timestamp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
	t_wdw AS (PARTITION BY station_id ORDER BY "temp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
	w_wdw AS (PARTITION BY station_id ORDER BY "wspdx" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
ON CONFLICT (station_id, "timestamp") DO NOTHING
RETURNING *;

-- name: InsertCurrentMOObservations :many
INSERT INTO observations_current (
  station_id,
	rain, "temp", rh,
	wdir, wspd, srad, mslp,
	tn, tx, gust, rain_accum,
	tn_timestamp, tx_timestamp, gust_timestamp, "timestamp")
SELECT DISTINCT(station_id),
	LAST_VALUE(rr / 6) OVER wdw::real AS rain,
	LAST_VALUE("temp") OVER wdw::real AS "temp",
	LAST_VALUE(rh) OVER wdw::real AS rh,
	LAST_VALUE(wdir) OVER wdw::real AS wdir,
	LAST_VALUE(wspd) OVER wdw::real AS wspd,
	LAST_VALUE(srad) OVER wdw::real AS srad,
	LAST_VALUE(pres) OVER wdw::real AS mslp,
	FIRST_VALUE("temp") OVER t_wdw::real AS tn,
	LAST_VALUE("temp") OVER t_wdw::real AS tx,
	LAST_VALUE("wspdx") OVER w_wdw::real AS gust,
	SUM(rr / 6) OVER t_wdw::real AS rain_accum,
	FIRST_VALUE("timestamp") OVER t_wdw::TIMESTAMPTZ AS tn_timestamp,
	LAST_VALUE("timestamp") OVER t_wdw::TIMESTAMPTZ AS tx_timestamp,
	LAST_VALUE("timestamp") OVER w_wdw::TIMESTAMPTZ AS gust_timestamp,
	LAST_VALUE("timestamp") OVER wdw::TIMESTAMPTZ AS "timestamp"
FROM observations_mo_observation
WHERE "timestamp" BETWEEN CURRENT_DATE AND CURRENT_TIMESTAMP
WINDOW
  wdw AS (PARTITION BY station_id ORDER BY "timestamp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
	t_wdw AS (PARTITION BY station_id ORDER BY "temp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
	w_wdw AS (PARTITION BY station_id ORDER BY "wspdx" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
ON CONFLICT (station_id, "timestamp") DO NOTHING
RETURNING *;

-- name: CreateCurrentObservation :one
INSERT INTO observations_current (
  station_id,
	rain, "temp", rh,
	wdir, wspd, srad, mslp,
	tn, tx, gust, rain_accum,
	tn_timestamp, tx_timestamp, gust_timestamp, "timestamp"
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;
