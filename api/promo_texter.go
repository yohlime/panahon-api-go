package api

import (
	"errors"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type lufftSMSReq struct {
	Number string `json:"number" binding:"required,mobile_number"`
	Msg    string `json:"msg" binding:"required"`
}

type lufftSMSRes struct {
	Station db.ObservationsStation       `json:"station"`
	Obs     db.ObservationsObservation   `json:"observation"`
	Health  db.ObservationsStationhealth `json:"health"`
}

// CreateLufftObservationHealth godoc
// @Summary      Store Lufft observation and health
// @Tags         Lufft
// @Produce      json
// @Success      200 {object} db.CreateStationObservationAndHealthTxResult
// @Router       /sm [post]
func (s *Server) CreateLufftObservationHealth(ctx *gin.Context) {
	var req lufftSMSReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).
			Msg("[SM] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	lufft, err := util.NewLufftFromString(req.Msg)
	if err != nil {
		log.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[SM] Invalid string")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	station, err := s.store.GetStationByMobileNumber(ctx, util.NullString{
		Text: pgtype.Text{
			String: req.Number,
			Valid:  true,
		},
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			log.Error().Err(err).
				Str("sender", req.Number).
				Str("msg", req.Msg).
				Msg("[SM] No station found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		log.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[SM] AN error occured")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	obsArg := db.CreateStationObservationParams{
		StationID: station.ID,
		Pres:      lufft.Obs.Pres,
		Rr:        lufft.Obs.Rr,
		Rh:        lufft.Obs.Rh,
		Temp:      lufft.Obs.Temp,
		Td:        lufft.Obs.Td,
		Wdir:      lufft.Obs.Wdir,
		Wspd:      lufft.Obs.Wspd,
		Wspdx:     lufft.Obs.Wspdx,
		Srad:      lufft.Obs.Srad,
		Mslp:      lufft.Obs.Mslp,
		Hi:        lufft.Obs.Hi,
		Wchill:    lufft.Obs.Wchill,
		Timestamp: lufft.Obs.Timestamp,
	}

	obs, err := s.store.CreateStationObservation(ctx, obsArg)
	if err != nil {
		log.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[SM] Cannot store station observation")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	healthArg := db.CreateStationHealthParams{
		StationID:         station.ID,
		Vb1:               lufft.Health.Vb1,
		Vb2:               lufft.Health.Vb2,
		Curr:              lufft.Health.Curr,
		Bp1:               lufft.Health.Bp1,
		Bp2:               lufft.Health.Bp2,
		Cm:                lufft.Health.Cm,
		Ss:                lufft.Health.Ss,
		TempArq:           lufft.Health.TempArq,
		RhArq:             lufft.Health.RhArq,
		Fpm:               lufft.Health.Fpm,
		MinutesDifference: lufft.Health.MinutesDifference,
		DataCount:         lufft.Health.DataCount,
		DataStatus:        lufft.Health.DataStatus,
		Timestamp:         lufft.Health.Timestamp,
		Message:           lufft.Health.Message,
		ErrorMsg:          lufft.Health.ErrorMsg,
	}

	health, err := s.store.CreateStationHealth(ctx, healthArg)
	if err != nil {
		log.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[SM] Cannot store station status")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	res := lufftSMSRes{
		Station: station,
		Obs:     obs,
		Health:  health,
	}

	log.Debug().Str("sender", req.Number).Msg("[SM] Data saved successfully")
	ctx.JSON(http.StatusCreated, res)
}
