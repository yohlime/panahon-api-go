package api

import (
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type stationHealthResponse struct {
	ID         int64              `json:"id"`
	Vb1        util.NullFloat4    `json:"vb1"`
	Vb2        util.NullFloat4    `json:"vb2"`
	Curr       util.NullFloat4    `json:"curr"`
	Bp1        util.NullFloat4    `json:"bp1"`
	Bp2        util.NullFloat4    `json:"bp2"`
	Cm         util.NullString    `json:"cm"`
	Ss         util.NullInt4      `json:"ss"`
	TempArq    util.NullFloat4    `json:"temp_arq"`
	RhArq      util.NullFloat4    `json:"rh_arq"`
	Fpm        util.NullString    `json:"fpm"`
	ErrorMsg   util.NullString    `json:"error_msg"`
	Message    util.NullString    `json:"message"`
	DataCount  util.NullInt4      `json:"data_count"`
	DataStatus util.NullString    `json:"data_status"`
	Timestamp  pgtype.Timestamptz `json:"timestamp"`
	StationID  int64              `json:"station_id"`
} //@name StationHealthResponse

func newStationHealthResponse(h db.ObservationsStationhealth) stationHealthResponse {
	return stationHealthResponse{
		ID:        h.ID,
		StationID: h.StationID,
		Vb1:       h.Vb1,
		Vb2:       h.Vb2,
		Curr:      h.Curr,
		Bp1:       h.Bp1,
		Bp2:       h.Bp2,
		Cm:        h.Cm,
		Ss:        h.Ss,
		TempArq:   h.TempArq,
		RhArq:     h.RhArq,
		Fpm:       h.Fpm,
		ErrorMsg:  h.ErrorMsg,
		Timestamp: h.Timestamp,
	}
}
