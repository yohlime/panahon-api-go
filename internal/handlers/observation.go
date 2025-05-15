package handlers

import (
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
)

type observationRes struct {
	Station models.Station            `json:"station"`
	Obs     models.StationObservation `json:"observation"`
	Health  StationHealth             `json:"health"`
} //@name ObservationResponse

func newObservationResponse(stn db.ObservationsStation, obs db.ObservationsObservation, h db.ObservationsStationhealth) observationRes {
	return observationRes{
		Station: models.NewStation(stn, false),
		Obs:     models.NewStationObservation(obs),
		Health:  newStationHealth(h),
	}
}
