package handlers

import (
	"errors"
	"fmt"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type pTexterStoreLufftReq struct {
	Number string `json:"number" binding:"required"`
	Msg    string `json:"msg" binding:"required"`
} //@name LufftSMSParams

// PromoTexterStoreLufft
//
//	@Summary	Store Lufft observation and health
//	@Tags		promotexter
//	@Accept		json
//	@Produce	json
//	@Param		req	query		pTexterStoreLufftReq	true	"Promo Texter query"
//	@Success	200	{object}	lufftRes
//	@Router		/ptexter [post]
func (h *DefaultHandler) PromoTexterStoreLufft(ctx *gin.Context) {
	var req pTexterStoreLufftReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).
			Msg("[PromoTexter] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	lufft, err := sensor.NewLufftFromString(req.Msg)
	if err != nil {
		h.logger.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[PromoTexter] Invalid string")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	mobileNumber, ok := util.ParseMobileNumber(req.Number)
	if !ok {
		err := fmt.Errorf("invalid mobile number: %s", req.Number)
		h.logger.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[PromoTexter] Invalid mobile number")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	station, err := h.store.GetStationByMobileNumber(ctx, pgtype.Text{
		String: mobileNumber,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			h.logger.Error().Err(err).
				Str("sender", req.Number).
				Str("msg", req.Msg).
				Msg("[PromoTexter] No station found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		h.logger.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[PromoTexter] AN error occured")
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

	obs, err := h.store.CreateStationObservation(ctx, obsArg)
	if err != nil {
		h.logger.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[PromoTexter] Cannot store station observation")
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

	health, err := h.store.CreateStationHealth(ctx, healthArg)
	if err != nil {
		h.logger.Error().Err(err).
			Str("sender", req.Number).
			Str("msg", req.Msg).
			Msg("[PromoTexter] Cannot store station status")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	res := newLufftResponse(station, obs, health)

	h.logger.Debug().Str("sender", req.Number).Msg("[PromoTexter] Data saved successfully")
	ctx.JSON(http.StatusCreated, res)
}
