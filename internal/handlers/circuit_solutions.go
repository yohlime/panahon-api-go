package handlers

import (
	"errors"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type csiStoreMisolReq struct {
	WeatherStr string `json:"weather" binding:"required"`
} //@name CSIMisolParams

// CSIStoreMisol
//
//	@Summary	Store Misol observation and health
//	@Tags		csi,misol
//	@Accept		json
//	@Produce	json
//	@Param		req	body		csiStoreMisolReq	true	"Circuit Solutions Misol parameters"
//	@Success	200	{object}	observationRes
//	@Router		/csi [post]
func (h *DefaultHandler) CSIStoreMisol(ctx *gin.Context) {
	var req csiStoreMisolReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).
			Msg("[CircuitSolutions] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	misol, err := sensor.NewMisolFromString(req.WeatherStr)
	if err != nil {
		h.logger.Error().Err(err).
			Str("msg", req.WeatherStr).
			Msg("[CircuitSolutions] Invalid string")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	mStn, err := h.store.GetMisolStation(ctx, misol.StnID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			h.logger.Error().Err(err).
				Int64("misolID", misol.StnID).
				Str("msg", req.WeatherStr).
				Msg("[CircuitSolutions] No station found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		h.logger.Error().Err(err).
			Int64("misolID", misol.StnID).
			Str("msg", req.WeatherStr).
			Msg("[CircuitSolutions] An error occured")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	stn, err := h.store.GetStation(ctx, mStn.StationID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			h.logger.Error().Err(err).
				Int64("id", mStn.StationID).
				Str("msg", req.WeatherStr).
				Msg("[CircuitSolutions] No station found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		h.logger.Error().Err(err).
			Int64("id", mStn.StationID).
			Str("msg", req.WeatherStr).
			Msg("[CircuitSolutions] An error occured")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	obsArg := db.CreateStationObservationParams{
		StationID: stn.ID,
		Pres:      util.ToFloat4(misol.Obs.Pres),
		Rr:        util.ToFloat4(misol.Obs.Rr),
		Rh:        util.ToFloat4(misol.Obs.Rh),
		Temp:      util.ToFloat4(misol.Obs.Temp),
		Td:        util.ToFloat4(misol.Obs.Td),
		Wdir:      util.ToFloat4(misol.Obs.Wdir),
		Wspd:      util.ToFloat4(misol.Obs.Wspd),
		Wspdx:     util.ToFloat4(misol.Obs.Wspdx),
		Srad:      util.ToFloat4(misol.Obs.Srad),
		Mslp:      util.ToFloat4(misol.Obs.Mslp),
		Hi:        util.ToFloat4(misol.Obs.Hi),
		Wchill:    util.ToFloat4(misol.Obs.Wchill),
		Timestamp: pgtype.Timestamptz{
			Time:  misol.Obs.Timestamp,
			Valid: true,
		},
	}

	obs, err := h.store.CreateStationObservation(ctx, obsArg)
	if err != nil {
		h.logger.Error().Err(err).
			Int64("id", stn.ID).
			Str("msg", req.WeatherStr).
			Msg("[CircuitSolutions] Cannot store station observation")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	healthArg := db.CreateStationHealthParams{
		StationID: stn.ID,
		Vb1:       util.ToFloat4(misol.Health.Vb1),
		Vb2:       util.ToFloat4(misol.Health.Vb2),
		Curr:      util.ToFloat4(misol.Health.Curr),
		Bp1:       util.ToFloat4(misol.Health.Bp1),
		Bp2:       util.ToFloat4(misol.Health.Bp2),
		// Cm:                util.ToPgText(misol.Health.Cm),
		// Ss:                util.ToInt4(misol.Health.Ss),
		TempArq: util.ToFloat4(misol.Health.TempArq),
		RhArq:   util.ToFloat4(misol.Health.RhArq),
		// Fpm:               util.ToPgText(misol.Health.Fpm),
		MinutesDifference: util.ToInt4(&misol.Health.MinutesDifference),
		DataCount:         util.ToInt4(&misol.Health.DataCount),
		DataStatus:        util.ToPgText(misol.Health.DataStatus),
		Timestamp: pgtype.Timestamptz{
			Time:  misol.Health.Timestamp,
			Valid: true,
		},
		Message:  util.ToPgText(misol.Health.Message),
		ErrorMsg: util.ToPgText(misol.Health.ErrorMsg),
	}

	health, err := h.store.CreateStationHealth(ctx, healthArg)
	if err != nil {
		h.logger.Error().Err(err).
			Int64("id", stn.ID).
			Str("msg", req.WeatherStr).
			Msg("[CircuitSolutions] Cannot store station status")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	res := newObservationResponse(stn, obs, health)

	h.logger.Debug().
		Int64("id", stn.ID).
		Int64("misolID", mStn.ID).
		Str("msg", req.WeatherStr).
		Msg("[CircuitSolutions] Data saved successfully")
	ctx.JSON(http.StatusCreated, res)
}
