package api

import (
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type lufftMsgLogUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

type lufftMsgLogReq struct {
	Page  int32 `form:"page,default=1" binding:"omitempty,min=1"`
	Limit int32 `form:"limit,default=5" binding:"omitempty,min=1,max=30"`
} //@name LufftMsgLogParams

type lufftMsgLogRes struct {
	Timestamp pgtype.Timestamptz `json:"timestamp"`
	Message   util.NullString    `json:"message"`
} //@name LufftMsgLogResponse

func newLufftMsgLoResponse(res db.ListLufftStationMsgRow) lufftMsgLogRes {
	return lufftMsgLogRes{
		Timestamp: res.Timestamp,
		Message:   res.Message,
	}
}

// LufftMsgLog godoc
//
//	@Summary	Lufft Message Logs
//	@Tags		lufft
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int				true	"Station ID"
//	@Param		req			query	lufftMsgLogReq	false	"Lufft Message log query"
//	@Success	200			{array}	lufftMsgLogRes
//	@Router		/lufft/{station_id}/logs [get]
func (s *Server) LufftMsgLog(ctx *gin.Context) {
	var uri lufftMsgLogUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req lufftMsgLogReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.Limit
	arg := db.ListLufftStationMsgParams{
		StationID: uri.StationID,
		Limit:     req.Limit,
		Offset:    offset,
	}

	msgs, err := s.store.ListLufftStationMsg(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numMsg := len(msgs)
	if numMsg <= 0 {
		ctx.JSON(http.StatusOK, nil)
	}

	res := make([]lufftMsgLogRes, numMsg)
	for m, msg := range msgs {
		res[m] = newLufftMsgLoResponse(msg)
	}

	ctx.JSON(http.StatusOK, res)
}

type lufftRes struct {
	Station stationResponse       `json:"station"`
	Obs     stationObsResponse    `json:"observation"`
	Health  stationHealthResponse `json:"health"`
} //@name LufftResponse

func newLufftResponse(stn db.ObservationsStation, obs db.ObservationsObservation, h db.ObservationsStationhealth) lufftRes {
	return lufftRes{
		Station: newStationResponse(stn),
		Obs:     newStationObsResponse(obs),
		Health:  newStationHealthResponse(h),
	}
}
