// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: mo_observation.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const countMOObservations = `-- name: CountMOObservations :one
SELECT count(*) FROM observations_mo_observation
WHERE station_id = ANY($1::bigint[])
  AND (CASE WHEN $2::bool THEN timestamp >= $3 ELSE TRUE END)
  AND (CASE WHEN $4::bool THEN timestamp <= $5 ELSE TRUE END)
`

type CountMOObservationsParams struct {
	StationIds  []int64            `json:"station_ids"`
	IsStartDate bool               `json:"is_start_date"`
	StartDate   pgtype.Timestamptz `json:"start_date"`
	IsEndDate   bool               `json:"is_end_date"`
	EndDate     pgtype.Timestamptz `json:"end_date"`
}

func (q *Queries) CountMOObservations(ctx context.Context, arg CountMOObservationsParams) (int64, error) {
	row := q.db.QueryRow(ctx, countMOObservations,
		arg.StationIds,
		arg.IsStartDate,
		arg.StartDate,
		arg.IsEndDate,
		arg.EndDate,
	)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countStationMOObservations = `-- name: CountStationMOObservations :one
SELECT count(*) FROM observations_mo_observation
WHERE station_id = $1
  AND (CASE WHEN $2::bool THEN timestamp >= $3 ELSE TRUE END)
  AND (CASE WHEN $4::bool THEN timestamp <= $5 ELSE TRUE END)
`

type CountStationMOObservationsParams struct {
	StationID   int64              `json:"station_id"`
	IsStartDate bool               `json:"is_start_date"`
	StartDate   pgtype.Timestamptz `json:"start_date"`
	IsEndDate   bool               `json:"is_end_date"`
	EndDate     pgtype.Timestamptz `json:"end_date"`
}

func (q *Queries) CountStationMOObservations(ctx context.Context, arg CountStationMOObservationsParams) (int64, error) {
	row := q.db.QueryRow(ctx, countStationMOObservations,
		arg.StationID,
		arg.IsStartDate,
		arg.StartDate,
		arg.IsEndDate,
		arg.EndDate,
	)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createStationMOObservation = `-- name: CreateStationMOObservation :one
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
) RETURNING id, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx, created_at, updated_at
`

type CreateStationMOObservationParams struct {
	Pres      pgtype.Float4      `json:"pres"`
	Rr        pgtype.Float4      `json:"rr"`
	Rh        pgtype.Float4      `json:"rh"`
	Temp      pgtype.Float4      `json:"temp"`
	Td        pgtype.Float4      `json:"td"`
	Wdir      pgtype.Float4      `json:"wdir"`
	Wspd      pgtype.Float4      `json:"wspd"`
	Wspdx     pgtype.Float4      `json:"wspdx"`
	Srad      pgtype.Float4      `json:"srad"`
	Hi        pgtype.Float4      `json:"hi"`
	Wchill    pgtype.Float4      `json:"wchill"`
	Timestamp pgtype.Timestamptz `json:"timestamp"`
	QcLevel   int32              `json:"qc_level"`
	StationID int64              `json:"station_id"`
}

func (q *Queries) CreateStationMOObservation(ctx context.Context, arg CreateStationMOObservationParams) (ObservationsMoObservation, error) {
	row := q.db.QueryRow(ctx, createStationMOObservation,
		arg.Pres,
		arg.Rr,
		arg.Rh,
		arg.Temp,
		arg.Td,
		arg.Wdir,
		arg.Wspd,
		arg.Wspdx,
		arg.Srad,
		arg.Hi,
		arg.Wchill,
		arg.Timestamp,
		arg.QcLevel,
		arg.StationID,
	)
	var i ObservationsMoObservation
	err := row.Scan(
		&i.ID,
		&i.Pres,
		&i.Rr,
		&i.Rh,
		&i.Temp,
		&i.Td,
		&i.Wdir,
		&i.Wspd,
		&i.Wspdx,
		&i.Srad,
		&i.Hi,
		&i.StationID,
		&i.Timestamp,
		&i.Wchill,
		&i.Rain,
		&i.Tx,
		&i.Tn,
		&i.Wrun,
		&i.Thwi,
		&i.Thswi,
		&i.Senergy,
		&i.Sradx,
		&i.Uvi,
		&i.Uvdose,
		&i.Uvx,
		&i.Hdd,
		&i.Cdd,
		&i.Et,
		&i.QcLevel,
		&i.Wdirx,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteStationMOObservation = `-- name: DeleteStationMOObservation :exec
DELETE FROM observations_mo_observation WHERE station_id = $1 AND id = $2
`

type DeleteStationMOObservationParams struct {
	StationID int64 `json:"station_id"`
	ID        int64 `json:"id"`
}

func (q *Queries) DeleteStationMOObservation(ctx context.Context, arg DeleteStationMOObservationParams) error {
	_, err := q.db.Exec(ctx, deleteStationMOObservation, arg.StationID, arg.ID)
	return err
}

const getStationMOObservation = `-- name: GetStationMOObservation :one
SELECT id, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx, created_at, updated_at FROM observations_mo_observation
WHERE station_id = $1 AND id = $2 LIMIT 1
`

type GetStationMOObservationParams struct {
	StationID int64 `json:"station_id"`
	ID        int64 `json:"id"`
}

func (q *Queries) GetStationMOObservation(ctx context.Context, arg GetStationMOObservationParams) (ObservationsMoObservation, error) {
	row := q.db.QueryRow(ctx, getStationMOObservation, arg.StationID, arg.ID)
	var i ObservationsMoObservation
	err := row.Scan(
		&i.ID,
		&i.Pres,
		&i.Rr,
		&i.Rh,
		&i.Temp,
		&i.Td,
		&i.Wdir,
		&i.Wspd,
		&i.Wspdx,
		&i.Srad,
		&i.Hi,
		&i.StationID,
		&i.Timestamp,
		&i.Wchill,
		&i.Rain,
		&i.Tx,
		&i.Tn,
		&i.Wrun,
		&i.Thwi,
		&i.Thswi,
		&i.Senergy,
		&i.Sradx,
		&i.Uvi,
		&i.Uvdose,
		&i.Uvx,
		&i.Hdd,
		&i.Cdd,
		&i.Et,
		&i.QcLevel,
		&i.Wdirx,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listMOObservations = `-- name: ListMOObservations :many
SELECT id, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx, created_at, updated_at FROM observations_mo_observation
WHERE station_id = ANY($1::bigint[])
  AND (CASE WHEN $2::bool THEN timestamp >= $3 ELSE TRUE END)
  AND (CASE WHEN $4::bool THEN timestamp <= $5 ELSE TRUE END)
ORDER BY timestamp DESC
LIMIT $7
OFFSET $6
`

type ListMOObservationsParams struct {
	StationIds  []int64            `json:"station_ids"`
	IsStartDate bool               `json:"is_start_date"`
	StartDate   pgtype.Timestamptz `json:"start_date"`
	IsEndDate   bool               `json:"is_end_date"`
	EndDate     pgtype.Timestamptz `json:"end_date"`
	Offset      int32              `json:"offset"`
	Limit       pgtype.Int4        `json:"limit"`
}

func (q *Queries) ListMOObservations(ctx context.Context, arg ListMOObservationsParams) ([]ObservationsMoObservation, error) {
	rows, err := q.db.Query(ctx, listMOObservations,
		arg.StationIds,
		arg.IsStartDate,
		arg.StartDate,
		arg.IsEndDate,
		arg.EndDate,
		arg.Offset,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ObservationsMoObservation{}
	for rows.Next() {
		var i ObservationsMoObservation
		if err := rows.Scan(
			&i.ID,
			&i.Pres,
			&i.Rr,
			&i.Rh,
			&i.Temp,
			&i.Td,
			&i.Wdir,
			&i.Wspd,
			&i.Wspdx,
			&i.Srad,
			&i.Hi,
			&i.StationID,
			&i.Timestamp,
			&i.Wchill,
			&i.Rain,
			&i.Tx,
			&i.Tn,
			&i.Wrun,
			&i.Thwi,
			&i.Thswi,
			&i.Senergy,
			&i.Sradx,
			&i.Uvi,
			&i.Uvdose,
			&i.Uvx,
			&i.Hdd,
			&i.Cdd,
			&i.Et,
			&i.QcLevel,
			&i.Wdirx,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const listStationMOObservations = `-- name: ListStationMOObservations :many
SELECT id, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx, created_at, updated_at FROM observations_mo_observation
WHERE station_id = $1
  AND (CASE WHEN $2::bool THEN timestamp >= $3 ELSE TRUE END)
  AND (CASE WHEN $4::bool THEN timestamp <= $5 ELSE TRUE END)
ORDER BY timestamp DESC
LIMIT $7
OFFSET $6
`

type ListStationMOObservationsParams struct {
	StationID   int64              `json:"station_id"`
	IsStartDate bool               `json:"is_start_date"`
	StartDate   pgtype.Timestamptz `json:"start_date"`
	IsEndDate   bool               `json:"is_end_date"`
	EndDate     pgtype.Timestamptz `json:"end_date"`
	Offset      int32              `json:"offset"`
	Limit       pgtype.Int4        `json:"limit"`
}

func (q *Queries) ListStationMOObservations(ctx context.Context, arg ListStationMOObservationsParams) ([]ObservationsMoObservation, error) {
	rows, err := q.db.Query(ctx, listStationMOObservations,
		arg.StationID,
		arg.IsStartDate,
		arg.StartDate,
		arg.IsEndDate,
		arg.EndDate,
		arg.Offset,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ObservationsMoObservation{}
	for rows.Next() {
		var i ObservationsMoObservation
		if err := rows.Scan(
			&i.ID,
			&i.Pres,
			&i.Rr,
			&i.Rh,
			&i.Temp,
			&i.Td,
			&i.Wdir,
			&i.Wspd,
			&i.Wspdx,
			&i.Srad,
			&i.Hi,
			&i.StationID,
			&i.Timestamp,
			&i.Wchill,
			&i.Rain,
			&i.Tx,
			&i.Tn,
			&i.Wrun,
			&i.Thwi,
			&i.Thswi,
			&i.Senergy,
			&i.Sradx,
			&i.Uvi,
			&i.Uvdose,
			&i.Uvx,
			&i.Hdd,
			&i.Cdd,
			&i.Et,
			&i.QcLevel,
			&i.Wdirx,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const updateStationMOObservation = `-- name: UpdateStationMOObservation :one
UPDATE observations_mo_observation
SET
  pres = COALESCE($1, pres),
  rr = COALESCE($2, rr),
  rh = COALESCE($3, rh),
  temp = COALESCE($4, temp),
  td = COALESCE($5, td),
  wdir = COALESCE($6, wdir),
  wspd = COALESCE($7, wspd),
  wspdx = COALESCE($8, wspdx),
  srad = COALESCE($9, srad),
  hi = COALESCE($10, hi),
  wchill = COALESCE($11, wchill),
  timestamp = COALESCE($12, timestamp),
  qc_level = COALESCE($13, qc_level),
  updated_at = now()
WHERE station_id = $14 AND id = $15
RETURNING id, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx, created_at, updated_at
`

type UpdateStationMOObservationParams struct {
	Pres      pgtype.Float4      `json:"pres"`
	Rr        pgtype.Float4      `json:"rr"`
	Rh        pgtype.Float4      `json:"rh"`
	Temp      pgtype.Float4      `json:"temp"`
	Td        pgtype.Float4      `json:"td"`
	Wdir      pgtype.Float4      `json:"wdir"`
	Wspd      pgtype.Float4      `json:"wspd"`
	Wspdx     pgtype.Float4      `json:"wspdx"`
	Srad      pgtype.Float4      `json:"srad"`
	Hi        pgtype.Float4      `json:"hi"`
	Wchill    pgtype.Float4      `json:"wchill"`
	Timestamp pgtype.Timestamptz `json:"timestamp"`
	QcLevel   pgtype.Int4        `json:"qc_level"`
	StationID int64              `json:"station_id"`
	ID        int64              `json:"id"`
}

func (q *Queries) UpdateStationMOObservation(ctx context.Context, arg UpdateStationMOObservationParams) (ObservationsMoObservation, error) {
	row := q.db.QueryRow(ctx, updateStationMOObservation,
		arg.Pres,
		arg.Rr,
		arg.Rh,
		arg.Temp,
		arg.Td,
		arg.Wdir,
		arg.Wspd,
		arg.Wspdx,
		arg.Srad,
		arg.Hi,
		arg.Wchill,
		arg.Timestamp,
		arg.QcLevel,
		arg.StationID,
		arg.ID,
	)
	var i ObservationsMoObservation
	err := row.Scan(
		&i.ID,
		&i.Pres,
		&i.Rr,
		&i.Rh,
		&i.Temp,
		&i.Td,
		&i.Wdir,
		&i.Wspd,
		&i.Wspdx,
		&i.Srad,
		&i.Hi,
		&i.StationID,
		&i.Timestamp,
		&i.Wchill,
		&i.Rain,
		&i.Tx,
		&i.Tn,
		&i.Wrun,
		&i.Thwi,
		&i.Thswi,
		&i.Senergy,
		&i.Sradx,
		&i.Uvi,
		&i.Uvdose,
		&i.Uvx,
		&i.Hdd,
		&i.Cdd,
		&i.Et,
		&i.QcLevel,
		&i.Wdirx,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
