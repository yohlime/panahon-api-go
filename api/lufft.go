package api

import (
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/gin-gonic/gin"
)

type lufftMsgLogUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

type lufftMsgLogReq struct {
	Page  int32 `form:"page,default=1" binding:"omitempty,min=1"`
	Limit int32 `form:"limit,default=5" binding:"omitempty,min=1,max=30"`
}

// LufftMsgLog godoc
// @Summary      Lufft Message Logs
// @Tags         Lufft
// @Produce      json
// @Success      200 {object} []db.ListLufftStationMsgRow
// @Router       /lufft/{station_id}/logs [get]
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

	ctx.JSON(http.StatusOK, msgs)
}
