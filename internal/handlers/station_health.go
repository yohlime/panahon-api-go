package handlers

import (
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type StationHealth struct {
	ID         int64              `json:"id"`
	Vb1        float32            `json:"vb1"`
	Vb2        float32            `json:"vb2"`
	Curr       float32            `json:"curr"`
	Bp1        float32            `json:"bp1"`
	Bp2        float32            `json:"bp2"`
	Cm         string             `json:"cm"`
	Ss         int32              `json:"ss"`
	RhArq      float32            `json:"rh_arq"`
	TempArq    float32            `json:"temp_arq"`
	Fpm        string             `json:"fpm"`
	ErrorMsg   string             `json:"error_msg"`
	Message    string             `json:"message"`
	DataCount  int32              `json:"data_count"`
	DataStatus string             `json:"data_status"`
	Timestamp  pgtype.Timestamptz `json:"timestamp"`
	StationID  int64              `json:"station_id"`
} //@name StationHealth

func newStationHealth(h db.ObservationsStationhealth) StationHealth {
	res := StationHealth{
		ID:        h.ID,
		StationID: h.StationID,
		Timestamp: h.Timestamp,
	}

	if h.Vb1.Valid {
		res.Vb1 = h.Vb1.Float32
	}
	if h.Vb2.Valid {
		res.Vb2 = h.Vb2.Float32
	}
	if h.Curr.Valid {
		res.Curr = h.Curr.Float32
	}
	if h.Bp1.Valid {
		res.Bp1 = h.Bp1.Float32
	}
	if h.Bp2.Valid {
		res.Bp2 = h.Bp2.Float32
	}
	if h.TempArq.Valid {
		res.TempArq = h.TempArq.Float32
	}
	if h.RhArq.Valid {
		res.RhArq = h.RhArq.Float32
	}
	if h.Ss.Valid {
		res.Ss = h.Ss.Int32
	}
	if h.Cm.Valid {
		res.Cm = h.Cm.String
	}
	if h.Fpm.Valid {
		res.Fpm = h.Fpm.String
	}
	if h.ErrorMsg.Valid {
		res.ErrorMsg = h.ErrorMsg.String
	}
	return res
}
