package api

import (
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type lufftMsgLogUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

type lufftMsgLogReq struct {
	Page    int32 `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int32 `form:"per_page,default=5" binding:"omitempty,min=1,max=30"`
} //@name LufftMsgLogParams

type lufftMsgLog struct {
	Timestamp pgtype.Timestamptz `json:"timestamp"`
	Message   pgtype.Text        `json:"message"`
} //@name LufftMsgLog

func newLufftMsgLoResponse(res db.ListLufftStationMsgRow) lufftMsgLog {
	return lufftMsgLog{
		Timestamp: res.Timestamp,
		Message:   res.Message,
	}
}

type paginatedLufftMsgLogs = util.PaginatedList[lufftMsgLog]

// LufftMsgLog
//
//	@Summary	Lufft Message Logs
//	@Tags		lufft
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path		int				true	"Station ID"
//	@Param		req			query		lufftMsgLogReq	false	"Lufft Message log query"
//	@Success	200			{object}	paginatedLufftMsgLogs
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

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListLufftStationMsgParams{
		StationID: uri.StationID,
		Limit:     req.PerPage,
		Offset:    offset,
	}

	msgs, err := s.store.ListLufftStationMsg(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numMsg := len(msgs)
	items := make([]lufftMsgLog, numMsg)
	for m, msg := range msgs {
		items[m] = newLufftMsgLoResponse(msg)
	}

	count, err := s.store.CountLufftStationMsg(ctx, uri.StationID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList[lufftMsgLog](req.Page, req.PerPage, int32(count), items)

	ctx.JSON(http.StatusOK, res)
}

type lufftRes struct {
	Station Station            `json:"station"`
	Obs     StationObservation `json:"observation"`
	Health  StationHealth      `json:"health"`
} //@name LufftResponse

func newLufftResponse(stn db.ObservationsStation, obs db.ObservationsObservation, h db.ObservationsStationhealth) lufftRes {
	return lufftRes{
		Station: newStation(stn),
		Obs:     newStationObservation(obs),
		Health:  newStationHealth(h),
	}
}
