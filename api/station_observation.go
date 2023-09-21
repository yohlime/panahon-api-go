package api

import (
	"errors"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type stationObsResponse struct {
	ID        int64              `json:"id"`
	Pres      util.NullFloat4    `json:"pres"`
	Rr        util.NullFloat4    `json:"rr"`
	Rh        util.NullFloat4    `json:"rh"`
	Temp      util.NullFloat4    `json:"temp"`
	Td        util.NullFloat4    `json:"td"`
	Wdir      util.NullFloat4    `json:"wdir"`
	Wspd      util.NullFloat4    `json:"wspd"`
	Wspdx     util.NullFloat4    `json:"wspdx"`
	Srad      util.NullFloat4    `json:"srad"`
	Mslp      util.NullFloat4    `json:"mslp"`
	Hi        util.NullFloat4    `json:"hi"`
	StationID int64              `json:"station_id"`
	Timestamp pgtype.Timestamptz `json:"timestamp"`
	Wchill    util.NullFloat4    `json:"wchill"`
	QcLevel   int32              `json:"qc_level"`
} //@name StationObservationResponse

func newStationObsResponse(obs db.ObservationsObservation) stationObsResponse {
	return stationObsResponse{
		ID:        obs.ID,
		StationID: obs.StationID,
		Pres:      obs.Pres,
		Rr:        obs.Rr,
		Rh:        obs.Rh,
		Temp:      obs.Temp,
		Td:        obs.Td,
		Wdir:      obs.Wdir,
		Wspd:      obs.Wspd,
		Wspdx:     obs.Wspdx,
		Srad:      obs.Srad,
		Mslp:      obs.Mslp,
		Hi:        obs.Hi,
		Wchill:    obs.Wchill,
		Timestamp: obs.Timestamp,
		QcLevel:   obs.QcLevel,
	}
}

type listStationObsUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

type listStationObsReq struct {
	Page    int32 `form:"page,default=1" binding:"omitempty,min=1"`            // page number
	PerPage int32 `form:"per_page,default=5" binding:"omitempty,min=1,max=30"` // limit
} //name ListStationObservationsParams

type listStationObsRes struct {
	Page    int32                `json:"page"`
	PerPage int32                `json:"per_page"`
	Total   int64                `json:"total"`
	Data    []stationObsResponse `json:"data"`
} //@name ListStationObservationsResponse

// ListStationObservations
//
//	@Summary	List station observations
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		req	query		listStationObsReq	false	"List station observations parameters"
//	@Success	200	{object}	listStationObsRes
//	@Router		/stations/{station_id}/observations [get]
func (s *Server) ListStationObservations(ctx *gin.Context) {
	var uri listStationObsUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req listStationObsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListStationObservationsParams{
		StationID: uri.StationID,
		Limit: util.NullInt4{
			Int4: pgtype.Int4{
				Int32: req.PerPage,
				Valid: true,
			},
		},
		Offset: offset,
	}

	observations, err := s.store.ListStationObservations(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numObs := len(observations)
	obsRes := make([]stationObsResponse, numObs)
	for i, observation := range observations {
		obsRes[i] = newStationObsResponse(observation)
	}

	totalObs, err := s.store.CountStationObservations(ctx, uri.StationID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := listStationObsRes{
		Page:    req.Page,
		PerPage: req.PerPage,
		Total:   totalObs,
		Data:    obsRes,
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getStationObsReq struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
	ID        int64 `uri:"id" binding:"required,min=1"`
}

// GetStationObservation
//
//	@Summary	Get station observation
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path		int	true	"Station ID"
//	@Param		id			path		int	true	"Station Observation ID"
//	@Success	200			{object}	stationObsResponse
//	@Router		/stations/{station_id}/observations/{id} [get]
func (s *Server) GetStationObservation(ctx *gin.Context) {
	var req getStationObsReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.GetStationObservationParams{
		StationID: req.StationID,
		ID:        req.ID,
	}

	obs, err := s.store.GetStationObservation(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station observation not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newStationObsResponse(obs))
}

type createStationObsUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

type createStationObsReq struct {
	Pres      util.NullFloat4    `json:"pres" binding:"omitempty,numeric"`
	Rr        util.NullFloat4    `json:"rr" binding:"omitempty,numeric"`
	Rh        util.NullFloat4    `json:"rh" binding:"omitempty,numeric"`
	Temp      util.NullFloat4    `json:"temp" binding:"omitempty,numeric"`
	Td        util.NullFloat4    `json:"td" binding:"omitempty,numeric"`
	Wdir      util.NullFloat4    `json:"wdir" binding:"omitempty,numeric"`
	Wspd      util.NullFloat4    `json:"wspd" binding:"omitempty,numeric"`
	Wspdx     util.NullFloat4    `json:"wspdx" binding:"omitempty,numeric"`
	Srad      util.NullFloat4    `json:"srad" binding:"omitempty,numeric"`
	Mslp      util.NullFloat4    `json:"mslp" binding:"omitempty,numeric"`
	Hi        util.NullFloat4    `json:"hi" binding:"omitempty,numeric"`
	Wchill    util.NullFloat4    `json:"wchill" binding:"omitempty,numeric"`
	QcLevel   int32              `json:"qc_level" binding:"omitempty,numeric"`
	Timestamp pgtype.Timestamptz `json:"timestamp" binding:"omitempty,numeric"`
} //@name CreateStationObservationParams

// CreateStationObservation
//
//	@Summary	Create station observation
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int					true	"Station ID"
//	@Param		stnObs		body	createStationObsReq	true	"Create station observation parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	stationObsResponse
//	@Router		/stations/{station_id}/observations [post]
func (s *Server) CreateStationObservation(ctx *gin.Context) {
	var uri createStationObsUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req createStationObsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateStationObservationParams{
		StationID: uri.StationID,
		Pres:      req.Pres,
		Rr:        req.Rr,
		Rh:        req.Rh,
		Temp:      req.Temp,
		Td:        req.Td,
		Wdir:      req.Wdir,
		Wspd:      req.Wspd,
		Wspdx:     req.Wspdx,
		Srad:      req.Srad,
		Mslp:      req.Mslp,
		Hi:        req.Hi,
		Wchill:    req.Wchill,
		Timestamp: req.Timestamp,
		QcLevel:   req.QcLevel,
	}

	obs, err := s.store.CreateStationObservation(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newStationObsResponse(obs))
}

type updateStationObsUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
	ID        int64 `uri:"id" binding:"required,min=1"`
}

type updateStationObsReq struct {
	Pres      util.NullFloat4    `json:"pres" binding:"omitempty,numeric"`
	Rr        util.NullFloat4    `json:"rr" binding:"omitempty,numeric"`
	Rh        util.NullFloat4    `json:"rh" binding:"omitempty,numeric"`
	Temp      util.NullFloat4    `json:"temp" binding:"omitempty,numeric"`
	Td        util.NullFloat4    `json:"td" binding:"omitempty,numeric"`
	Wdir      util.NullFloat4    `json:"wdir" binding:"omitempty,numeric"`
	Wspd      util.NullFloat4    `json:"wspd" binding:"omitempty,numeric"`
	Wspdx     util.NullFloat4    `json:"wspdx" binding:"omitempty,numeric"`
	Srad      util.NullFloat4    `json:"srad" binding:"omitempty,numeric"`
	Mslp      util.NullFloat4    `json:"mslp" binding:"omitempty,numeric"`
	Hi        util.NullFloat4    `json:"hi" binding:"omitempty,numeric"`
	Wchill    util.NullFloat4    `json:"wchill" binding:"omitempty,numeric"`
	QcLevel   util.NullInt4      `json:"qc_level" binding:"omitempty,numeric"`
	Timestamp pgtype.Timestamptz `json:"timestamp" binding:"omitempty,numeric"`
} //@name UpdateStationObservationParams

// UpdateStationObservation
//
//	@Summary	Update station observation
//	@Tags		observations
//	@Produce	json
//	@Param		station_id	path	int					true	"Station ID"
//	@Param		id			path	int					true	"Station Observation ID"
//	@Param		stnObs		body	updateStationObsReq	true	"Update station observation parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	stationObsResponse
//	@Router		/stations/{station_id}/observations/{id} [put]
func (s *Server) UpdateStationObservation(ctx *gin.Context) {
	var uri updateStationObsUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateStationObsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.UpdateStationObservationParams{
		ID:        uri.ID,
		StationID: uri.StationID,
		Pres:      req.Pres,
		Rr:        req.Rr,
		Rh:        req.Rh,
		Temp:      req.Temp,
		Td:        req.Td,
		Wdir:      req.Wdir,
		Wspd:      req.Wspd,
		Wspdx:     req.Wspdx,
		Srad:      req.Srad,
		Mslp:      req.Mslp,
		Hi:        req.Hi,
		Wchill:    req.Wchill,
		Timestamp: req.Timestamp,
		QcLevel:   req.QcLevel,
	}

	obs, err := s.store.UpdateStationObservation(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newStationObsResponse(obs))
}

type deleteStationObsReq struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
	ID        int64 `uri:"id" binding:"required,min=1"`
}

// DeleteStationObservation
//
//	@Summary	Delete station observation
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int	true	"Station ID"
//	@Param		id			path	int	true	"Station Observation ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/stations/{station_id}/observations/{id} [delete]
func (s *Server) DeleteStationObservation(ctx *gin.Context) {
	var req deleteStationObsReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.DeleteStationObservationParams{
		ID:        req.ID,
		StationID: req.StationID,
	}

	err := s.store.DeleteStationObservation(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
