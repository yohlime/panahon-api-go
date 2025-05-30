// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: observations_current.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createCurrentObservation = `-- name: CreateCurrentObservation :one
INSERT INTO observations_current (
  station_id,
	rain, "temp", rh,
	wdir, wspd, srad, mslp,
	tn, tx, gust, rain_accum,
	tn_timestamp, tx_timestamp, gust_timestamp, "timestamp"
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING id, station_id, rain, temp, rh, wdir, wspd, srad, mslp, tn, tx, gust, rain_accum, timestamp, tn_timestamp, tx_timestamp, gust_timestamp
`

type CreateCurrentObservationParams struct {
	StationID     int64              `json:"station_id"`
	Rain          pgtype.Float4      `json:"rain"`
	Temp          pgtype.Float4      `json:"temp"`
	Rh            pgtype.Float4      `json:"rh"`
	Wdir          pgtype.Float4      `json:"wdir"`
	Wspd          pgtype.Float4      `json:"wspd"`
	Srad          pgtype.Float4      `json:"srad"`
	Mslp          pgtype.Float4      `json:"mslp"`
	Tn            pgtype.Float4      `json:"tn"`
	Tx            pgtype.Float4      `json:"tx"`
	Gust          pgtype.Float4      `json:"gust"`
	RainAccum     pgtype.Float4      `json:"rain_accum"`
	TnTimestamp   pgtype.Timestamptz `json:"tn_timestamp"`
	TxTimestamp   pgtype.Timestamptz `json:"tx_timestamp"`
	GustTimestamp pgtype.Timestamptz `json:"gust_timestamp"`
	Timestamp     pgtype.Timestamptz `json:"timestamp"`
}

func (q *Queries) CreateCurrentObservation(ctx context.Context, arg CreateCurrentObservationParams) (ObservationsCurrent, error) {
	row := q.db.QueryRow(ctx, createCurrentObservation,
		arg.StationID,
		arg.Rain,
		arg.Temp,
		arg.Rh,
		arg.Wdir,
		arg.Wspd,
		arg.Srad,
		arg.Mslp,
		arg.Tn,
		arg.Tx,
		arg.Gust,
		arg.RainAccum,
		arg.TnTimestamp,
		arg.TxTimestamp,
		arg.GustTimestamp,
		arg.Timestamp,
	)
	var i ObservationsCurrent
	err := row.Scan(
		&i.ID,
		&i.StationID,
		&i.Rain,
		&i.Temp,
		&i.Rh,
		&i.Wdir,
		&i.Wspd,
		&i.Srad,
		&i.Mslp,
		&i.Tn,
		&i.Tx,
		&i.Gust,
		&i.RainAccum,
		&i.Timestamp,
		&i.TnTimestamp,
		&i.TxTimestamp,
		&i.GustTimestamp,
	)
	return i, err
}

const getLatestStationObservation = `-- name: GetLatestStationObservation :one
SELECT
  stn.id, stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
  obs.id, obs.station_id, obs.rain, obs.temp, obs.rh, obs.wdir, obs.wspd, obs.srad, obs.mslp, obs.tn, obs.tx, obs.gust, obs.rain_accum, obs.timestamp, obs.tn_timestamp, obs.tx_timestamp, obs.gust_timestamp
FROM observations_station stn 
  JOIN observations_current obs 
  ON stn.id = obs.station_id
WHERE stn.id = $1
ORDER BY obs.timestamp DESC
LIMIT 1
`

type GetLatestStationObservationRow struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"`
	Lat                 pgtype.Float4       `json:"lat"`
	Lon                 pgtype.Float4       `json:"lon"`
	Elevation           pgtype.Float4       `json:"elevation"`
	Address             pgtype.Text         `json:"address"`
	ObservationsCurrent ObservationsCurrent `json:"observations_current"`
}

func (q *Queries) GetLatestStationObservation(ctx context.Context, id int64) (GetLatestStationObservationRow, error) {
	row := q.db.QueryRow(ctx, getLatestStationObservation, id)
	var i GetLatestStationObservationRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Lat,
		&i.Lon,
		&i.Elevation,
		&i.Address,
		&i.ObservationsCurrent.ID,
		&i.ObservationsCurrent.StationID,
		&i.ObservationsCurrent.Rain,
		&i.ObservationsCurrent.Temp,
		&i.ObservationsCurrent.Rh,
		&i.ObservationsCurrent.Wdir,
		&i.ObservationsCurrent.Wspd,
		&i.ObservationsCurrent.Srad,
		&i.ObservationsCurrent.Mslp,
		&i.ObservationsCurrent.Tn,
		&i.ObservationsCurrent.Tx,
		&i.ObservationsCurrent.Gust,
		&i.ObservationsCurrent.RainAccum,
		&i.ObservationsCurrent.Timestamp,
		&i.ObservationsCurrent.TnTimestamp,
		&i.ObservationsCurrent.TxTimestamp,
		&i.ObservationsCurrent.GustTimestamp,
	)
	return i, err
}

const getNearestLatestStationObservation = `-- name: GetNearestLatestStationObservation :one
WITH NearestStation AS (
  SELECT
	id, name, lat, lon, elevation, address
  FROM observations_station
  ORDER BY geom <-> ST_Point($1::real, $2::real, 4326)
  LIMIT 1
)
SELECT
  stn.id, stn.name, stn.lat, stn.lon, stn.elevation, stn.address,
  obs.id, obs.station_id, obs.rain, obs.temp, obs.rh, obs.wdir, obs.wspd, obs.srad, obs.mslp, obs.tn, obs.tx, obs.gust, obs.rain_accum, obs.timestamp, obs.tn_timestamp, obs.tx_timestamp, obs.gust_timestamp
FROM NearestStation stn 
  JOIN observations_current obs 
  ON stn.id = obs.station_id
ORDER BY obs.timestamp DESC
LIMIT 1
`

type GetNearestLatestStationObservationParams struct {
	Lon float32 `json:"lon"`
	Lat float32 `json:"lat"`
}

type GetNearestLatestStationObservationRow struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"`
	Lat                 pgtype.Float4       `json:"lat"`
	Lon                 pgtype.Float4       `json:"lon"`
	Elevation           pgtype.Float4       `json:"elevation"`
	Address             pgtype.Text         `json:"address"`
	ObservationsCurrent ObservationsCurrent `json:"observations_current"`
}

func (q *Queries) GetNearestLatestStationObservation(ctx context.Context, arg GetNearestLatestStationObservationParams) (GetNearestLatestStationObservationRow, error) {
	row := q.db.QueryRow(ctx, getNearestLatestStationObservation, arg.Lon, arg.Lat)
	var i GetNearestLatestStationObservationRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Lat,
		&i.Lon,
		&i.Elevation,
		&i.Address,
		&i.ObservationsCurrent.ID,
		&i.ObservationsCurrent.StationID,
		&i.ObservationsCurrent.Rain,
		&i.ObservationsCurrent.Temp,
		&i.ObservationsCurrent.Rh,
		&i.ObservationsCurrent.Wdir,
		&i.ObservationsCurrent.Wspd,
		&i.ObservationsCurrent.Srad,
		&i.ObservationsCurrent.Mslp,
		&i.ObservationsCurrent.Tn,
		&i.ObservationsCurrent.Tx,
		&i.ObservationsCurrent.Gust,
		&i.ObservationsCurrent.RainAccum,
		&i.ObservationsCurrent.Timestamp,
		&i.ObservationsCurrent.TnTimestamp,
		&i.ObservationsCurrent.TxTimestamp,
		&i.ObservationsCurrent.GustTimestamp,
	)
	return i, err
}

const insertCurrentMOObservations = `-- name: InsertCurrentMOObservations :many
WITH "rounded_data" AS (
    SELECT
        id, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx, created_at, updated_at,
        TO_TIMESTAMP(ROUND(EXTRACT(EPOCH FROM "timestamp") / 600.0) * 600) AS rounded_ts
    FROM "observations_mo_observation"
    WHERE "timestamp" BETWEEN CURRENT_DATE AND CURRENT_TIMESTAMP
),

"agg_1d" AS (
    SELECT DISTINCT
        "station_id",
        FIRST_VALUE("temp") OVER "t_wdw" AS "tn",
        LAST_VALUE("temp") OVER "t_wdw" AS "tx",
        LAST_VALUE("wspdx") OVER "w_wdw" AS "gust",
        FIRST_VALUE("timestamp") OVER "t_wdw" AS "tn_timestamp",
        LAST_VALUE("timestamp") OVER "t_wdw" AS "tx_timestamp",
        LAST_VALUE("timestamp") OVER "w_wdw" AS "gust_timestamp",
	      LAST_VALUE("rounded_ts") OVER "wdw" AS "timestamp"
    FROM "rounded_data"
    WINDOW
        "wdw" AS (PARTITION BY "station_id" ORDER BY "timestamp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
        "t_wdw" AS (PARTITION BY "station_id" ORDER BY "temp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
        "w_wdw" AS (PARTITION BY "station_id" ORDER BY "wspdx" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
),

"agg_10m" AS (
    SELECT DISTINCT
        "station_id",
        LAST_VALUE("rr") OVER "wdw" AS "rain",
        SUM("rr" / 6) OVER "wdw" AS "rain_accum",
        LAST_VALUE("timestamp") OVER "wdw" AS "timestamp"
    FROM (
        SELECT
            "station_id",
	          "rounded_ts" AS "timestamp",
            AVG("rr") AS "rr"
        FROM "rounded_data"
        GROUP BY "station_id", "rounded_ts"
    ) bin_10m
    WINDOW "wdw" AS (PARTITION BY "station_id" ORDER BY "timestamp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
),

"other_t" AS (
    SELECT DISTINCT
        "station_id",
        LAST_VALUE("temp") OVER "wdw" AS "temp",
        LAST_VALUE("rh") OVER "wdw" AS "rh",
        LAST_VALUE("wdir") OVER "wdw" AS "wdir",
        LAST_VALUE("wspd") OVER "wdw" AS "wspd",
        LAST_VALUE("srad") OVER "wdw" AS "srad",
        LAST_VALUE("pres") OVER "wdw" AS "mslp",
        LAST_VALUE("rounded_ts") OVER "wdw" AS "timestamp"
    FROM "rounded_data"
    WINDOW "wdw" AS (PARTITION BY "station_id" ORDER BY "rounded_ts" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
)

INSERT INTO "observations_current" (
    "station_id",
    "rain", "temp", "rh",
    "wdir", "wspd", "srad", "mslp",
    "tn", "tx", "gust", "rain_accum",
    "tn_timestamp", "tx_timestamp", "gust_timestamp", "timestamp"
)
SELECT
    o.station_id,
    a10m.rain,
    o.temp, o.rh, o.wdir, o.wspd, o.srad, o.mslp,
    a1d.tn, a1d.tx, a1d.gust,
    a10m.rain_accum,
    a1d.tn_timestamp, a1d.tx_timestamp, a1d.gust_timestamp,
    o.timestamp
FROM agg_1d a1d
LEFT JOIN agg_10m a10m ON a1d.station_id = a10m.station_id AND a1d.timestamp = a10m.timestamp
LEFT JOIN other_t o ON o.station_id = a10m.station_id AND o.timestamp = a10m.timestamp
ON CONFLICT (station_id, "timestamp") DO NOTHING
RETURNING id, station_id, rain, temp, rh, wdir, wspd, srad, mslp, tn, tx, gust, rain_accum, timestamp, tn_timestamp, tx_timestamp, gust_timestamp
`

func (q *Queries) InsertCurrentMOObservations(ctx context.Context) ([]ObservationsCurrent, error) {
	rows, err := q.db.Query(ctx, insertCurrentMOObservations)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ObservationsCurrent{}
	for rows.Next() {
		var i ObservationsCurrent
		if err := rows.Scan(
			&i.ID,
			&i.StationID,
			&i.Rain,
			&i.Temp,
			&i.Rh,
			&i.Wdir,
			&i.Wspd,
			&i.Srad,
			&i.Mslp,
			&i.Tn,
			&i.Tx,
			&i.Gust,
			&i.RainAccum,
			&i.Timestamp,
			&i.TnTimestamp,
			&i.TxTimestamp,
			&i.GustTimestamp,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertCurrentObservations = `-- name: InsertCurrentObservations :many
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
RETURNING id, station_id, rain, temp, rh, wdir, wspd, srad, mslp, tn, tx, gust, rain_accum, timestamp, tn_timestamp, tx_timestamp, gust_timestamp
`

func (q *Queries) InsertCurrentObservations(ctx context.Context) ([]ObservationsCurrent, error) {
	rows, err := q.db.Query(ctx, insertCurrentObservations)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ObservationsCurrent{}
	for rows.Next() {
		var i ObservationsCurrent
		if err := rows.Scan(
			&i.ID,
			&i.StationID,
			&i.Rain,
			&i.Temp,
			&i.Rh,
			&i.Wdir,
			&i.Wspd,
			&i.Srad,
			&i.Mslp,
			&i.Tn,
			&i.Tx,
			&i.Gust,
			&i.RainAccum,
			&i.Timestamp,
			&i.TnTimestamp,
			&i.TxTimestamp,
			&i.GustTimestamp,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listLatestObservations = `-- name: ListLatestObservations :many
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
SELECT id, name, lat, lon, elevation, address, rain, temp, rh, wdir, wspd, srad, mslp, tn, tx, gust, rain_accum, tn_timestamp, tx_timestamp, gust_timestamp, timestamp, rn
FROM RankedRows
WHERE rn = 1
`

type ListLatestObservationsRow struct {
	ID            int64              `json:"id"`
	Name          string             `json:"name"`
	Lat           pgtype.Float4      `json:"lat"`
	Lon           pgtype.Float4      `json:"lon"`
	Elevation     pgtype.Float4      `json:"elevation"`
	Address       pgtype.Text        `json:"address"`
	Rain          pgtype.Float4      `json:"rain"`
	Temp          pgtype.Float4      `json:"temp"`
	Rh            pgtype.Float4      `json:"rh"`
	Wdir          pgtype.Float4      `json:"wdir"`
	Wspd          pgtype.Float4      `json:"wspd"`
	Srad          pgtype.Float4      `json:"srad"`
	Mslp          pgtype.Float4      `json:"mslp"`
	Tn            pgtype.Float4      `json:"tn"`
	Tx            pgtype.Float4      `json:"tx"`
	Gust          pgtype.Float4      `json:"gust"`
	RainAccum     pgtype.Float4      `json:"rain_accum"`
	TnTimestamp   pgtype.Timestamptz `json:"tn_timestamp"`
	TxTimestamp   pgtype.Timestamptz `json:"tx_timestamp"`
	GustTimestamp pgtype.Timestamptz `json:"gust_timestamp"`
	Timestamp     pgtype.Timestamptz `json:"timestamp"`
	Rn            int64              `json:"rn"`
}

func (q *Queries) ListLatestObservations(ctx context.Context) ([]ListLatestObservationsRow, error) {
	rows, err := q.db.Query(ctx, listLatestObservations)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListLatestObservationsRow{}
	for rows.Next() {
		var i ListLatestObservationsRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Lat,
			&i.Lon,
			&i.Elevation,
			&i.Address,
			&i.Rain,
			&i.Temp,
			&i.Rh,
			&i.Wdir,
			&i.Wspd,
			&i.Srad,
			&i.Mslp,
			&i.Tn,
			&i.Tx,
			&i.Gust,
			&i.RainAccum,
			&i.TnTimestamp,
			&i.TxTimestamp,
			&i.GustTimestamp,
			&i.Timestamp,
			&i.Rn,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
