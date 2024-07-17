package models

import (
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type BaseStationObs struct {
	Pres      *float32  `json:"pres" fake:"{float32range:990,1100}"`
	Rr        *float32  `json:"rr"`
	Rh        *float32  `json:"rh"`
	Temp      *float32  `json:"temp" fake:"{float32range:25,35}"`
	Td        *float32  `json:"td"`
	Wdir      *float32  `json:"wdir"`
	Wspd      *float32  `json:"wspd"`
	Wspdx     *float32  `json:"wspdx"`
	Srad      *float32  `json:"srad"`
	Mslp      *float32  `json:"mslp"`
	Hi        *float32  `json:"hi"`
	Wchill    *float32  `json:"wchill"`
	Timestamp time.Time `json:"timestamp"`
}

type StationObservation struct {
	ID        int64 `json:"id" fake:"{number:1,1000}"`
	StationID int64 `json:"station_id" fake:"{number:1,250}"`
	QcLevel   int32 `json:"qc_level"`
	BaseStationObs
} //@name StationObservation

// NewStationObservation creates new StationObservation from db.ObservationsObservation
func NewStationObservation(obs db.ObservationsObservation) StationObservation {
	res := StationObservation{
		ID:        obs.ID,
		StationID: obs.StationID,
		QcLevel:   obs.QcLevel,
	}

	if obs.Pres.Valid {
		res.Pres = &obs.Pres.Float32
	}
	if obs.Rr.Valid {
		res.Rr = &obs.Rr.Float32
	}
	if obs.Rh.Valid {
		res.Rh = &obs.Rh.Float32
	}
	if obs.Temp.Valid {
		res.Temp = &obs.Temp.Float32
	}
	if obs.Td.Valid {
		res.Td = &obs.Td.Float32
	}
	if obs.Wdir.Valid {
		res.Wdir = &obs.Wdir.Float32
	}
	if obs.Wspd.Valid {
		res.Wspd = &obs.Wspd.Float32
	}
	if obs.Wspdx.Valid {
		res.Wspdx = &obs.Wspdx.Float32
	}
	if obs.Srad.Valid {
		res.Srad = &obs.Srad.Float32
	}
	if obs.Mslp.Valid {
		res.Mslp = &obs.Mslp.Float32
	}
	if obs.Hi.Valid {
		res.Hi = &obs.Hi.Float32
	}
	if obs.Wchill.Valid {
		res.Wchill = &obs.Wchill.Float32
	}

	if obs.Timestamp.Valid {
		res.Timestamp = obs.Timestamp.Time
	}

	return res
}

type CreateStationObsReq struct {
	StationID int64 `json:"station_id"`
	QcLevel   int32 `json:"qc_level"`
	BaseStationObs
} //@name CreateStationObservationReq

func (r CreateStationObsReq) Transform() db.CreateStationObservationParams {
	return transformStationObs(
		r.BaseStationObs,
		db.CreateStationObservationParams{
			StationID: r.StationID,
			QcLevel:   r.QcLevel,
		})
}

type UpdateStationObsReq struct {
	ID        int64  `json:"id"`
	StationID int64  `json:"station_id"`
	QcLevel   *int32 `json:"qc_level"`
	BaseStationObs
} //@name UpdateStationObservationParams

func (r UpdateStationObsReq) Transform() db.UpdateStationObservationParams {
	return transformStationObs(
		r.BaseStationObs,
		db.UpdateStationObservationParams{
			ID:        r.ID,
			StationID: r.StationID,
			QcLevel:   util.ToInt4(r.QcLevel),
		})
}

type StationObsParams interface {
	db.CreateStationObservationParams | db.UpdateStationObservationParams
}

func transformStationObs[T StationObsParams](req BaseStationObs, extraParams T) T {
	arg := db.CreateStationObservationParams{
		Pres:   util.ToFloat4(req.Pres),
		Rr:     util.ToFloat4(req.Rr),
		Rh:     util.ToFloat4(req.Rh),
		Temp:   util.ToFloat4(req.Temp),
		Td:     util.ToFloat4(req.Td),
		Wdir:   util.ToFloat4(req.Wdir),
		Wspd:   util.ToFloat4(req.Wspd),
		Wspdx:  util.ToFloat4(req.Wspdx),
		Srad:   util.ToFloat4(req.Srad),
		Mslp:   util.ToFloat4(req.Mslp),
		Hi:     util.ToFloat4(req.Hi),
		Wchill: util.ToFloat4(req.Wchill),
		Timestamp: pgtype.Timestamptz{
			Time:  req.Timestamp,
			Valid: !req.Timestamp.IsZero(),
		},
	}

	switch v := any(extraParams).(type) {
	case db.CreateStationObservationParams:
		arg.StationID = v.StationID
		arg.QcLevel = v.QcLevel
		return any(arg).(T)
	case db.UpdateStationObservationParams:
		return any(db.UpdateStationObservationParams{
			ID:        v.ID,
			StationID: v.StationID,
			Pres:      arg.Pres,
			Rr:        arg.Rr,
			Rh:        arg.Rh,
			Temp:      arg.Temp,
			Td:        arg.Td,
			Wdir:      arg.Wdir,
			Wspd:      arg.Wspd,
			Wspdx:     arg.Wspdx,
			Srad:      arg.Srad,
			Mslp:      arg.Mslp,
			Hi:        arg.Hi,
			Wchill:    arg.Wchill,
			Timestamp: arg.Timestamp,
			QcLevel:   v.QcLevel,
		}).(T)
	default:
		panic("Unsupported type")
	}
}
