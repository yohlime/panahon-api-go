package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type createStationObsUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

// CreateStationObservation
//
//	@Summary	Create station observation
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int							true	"Station ID"
//	@Param		stnObs		body	models.CreateStationObsReq	true	"Create station observation parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	models.StationObservation
//	@Router		/stations/{station_id}/observations [post]
func (s *Server) CreateStationObservation(ctx *gin.Context) {
	var uri createStationObsUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.CreateStationObsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		fmt.Println(err)
		return
	}
	req.StationID = uri.StationID

	arg := req.Transform()

	obs, err := s.store.CreateStationObservation(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := models.NewStationObservation(obs)
	ctx.JSON(http.StatusCreated, res)
}

type listStationObsUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

type listStationObsReq struct {
	Page      int32  `form:"page,default=1" binding:"omitempty,min=1"`            // page number
	PerPage   int32  `form:"per_page,default=5" binding:"omitempty,min=1,max=30"` // limit
	StartDate string `form:"start_date" binding:"omitempty,date_time"`
	EndDate   string `form:"end_date" binding:"omitempty,date_time"`
} //@name ListStationObservationsParams

type paginatedStationObservations = util.PaginatedList[models.StationObservation] //@name PaginatedStationObservations

// ListStationObservations
//
//	@Summary	List station observations
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path		int					true	"Station ID"
//	@Param		req			query		listStationObsReq	false	"List station observations parameters"
//	@Success	200			{object}	paginatedStationObservations
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

	startDate, isStartDate := util.ParseDateTime(req.StartDate)
	endDate, isEndDate := util.ParseDateTime(req.EndDate)

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListStationObservationsParams{
		StationID: uri.StationID,
		Limit: pgtype.Int4{
			Int32: req.PerPage,
			Valid: true,
		},
		Offset:      offset,
		IsStartDate: isStartDate,
		StartDate: pgtype.Timestamptz{
			Time:  startDate,
			Valid: !startDate.IsZero(),
		},
		IsEndDate: isEndDate,
		EndDate: pgtype.Timestamptz{
			Time:  endDate,
			Valid: !endDate.IsZero(),
		},
	}

	observations, err := s.store.ListStationObservations(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numObs := len(observations)
	items := make([]models.StationObservation, numObs)
	for i, observation := range observations {
		items[i] = models.NewStationObservation(observation)
	}

	count, err := s.store.CountStationObservations(ctx, db.CountStationObservationsParams{
		StationID:   arg.StationID,
		IsStartDate: arg.IsStartDate,
		StartDate:   arg.StartDate,
		IsEndDate:   arg.IsEndDate,
		EndDate:     arg.EndDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList(req.Page, req.PerPage, int32(count), items)

	ctx.JSON(http.StatusOK, res)
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
//	@Success	200			{object}	models.StationObservation
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

	res := models.NewStationObservation(obs)
	ctx.JSON(http.StatusOK, res)
}

type updateStationObsUri struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
	ID        int64 `uri:"id" binding:"required,min=1"`
}

// UpdateStationObservation
//
//	@Summary	Update station observation
//	@Tags		observations
//	@Produce	json
//	@Param		station_id	path	int							true	"Station ID"
//	@Param		id			path	int							true	"Station Observation ID"
//	@Param		stnObs		body	models.UpdateStationObsReq	true	"Update station observation parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	models.StationObservation
//	@Router		/stations/{station_id}/observations/{id} [put]
func (s *Server) UpdateStationObservation(ctx *gin.Context) {
	var uri updateStationObsUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.UpdateStationObsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	req.ID = uri.ID
	req.StationID = uri.StationID

	arg := req.Transform()

	obs, err := s.store.UpdateStationObservation(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := models.NewStationObservation(obs)
	ctx.JSON(http.StatusOK, res)
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

type listObservationsReq struct {
	Page       int32  `form:"page,default=1" binding:"omitempty,min=1"`            // page number
	PerPage    int32  `form:"per_page,default=5" binding:"omitempty,min=1,max=30"` // limit
	StationIDs string `form:"station_ids" binding:"omitempty"`
	StartDate  string `form:"start_date" binding:"omitempty,date_time"`
	EndDate    string `form:"end_date" binding:"omitempty,date_time"`
} //@name ListObservationsParams

// ListObservations
//
//	@Summary	list station observation
//	@Tags		observations
//	@Produce	json
//	@Param		req	query		listObservationsReq	false	"List observations parameters"
//	@Success	200	{object}	paginatedStationObservations
//	@Router		/observations [get]
func (s *Server) ListObservations(ctx *gin.Context) {
	var req listObservationsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var stationIDs []int64
	if len(req.StationIDs) == 0 {
		stations, err := s.store.ListStations(ctx, db.ListStationsParams{
			Limit:  pgtype.Int4{Int32: 10, Valid: true},
			Offset: 0,
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		for i := range stations {
			stationIDs = append(stationIDs, stations[i].ID)
		}
	} else {
		stnIDStrs := strings.Split(req.StationIDs, ",")
		for i := range stnIDStrs {
			stnID, err := strconv.ParseInt(stnIDStrs[i], 10, 64)
			if err != nil {
				continue
			}
			stationIDs = append(stationIDs, stnID)
		}
	}

	startDate, isStartDate := util.ParseDateTime(req.StartDate)
	endDate, isEndDate := util.ParseDateTime(req.EndDate)

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListObservationsParams{
		StationIds: stationIDs,
		Limit: pgtype.Int4{
			Int32: req.PerPage,
			Valid: true,
		},
		Offset:      offset,
		IsStartDate: isStartDate,
		StartDate: pgtype.Timestamptz{
			Time:  startDate,
			Valid: !startDate.IsZero(),
		},
		IsEndDate: isEndDate,
		EndDate: pgtype.Timestamptz{
			Time:  endDate,
			Valid: !endDate.IsZero(),
		},
	}

	obs, err := s.store.ListObservations(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numObs := len(obs)
	items := make([]models.StationObservation, numObs)
	for i, observation := range obs {
		items[i] = models.NewStationObservation(observation)
	}

	count, err := s.store.CountObservations(ctx, db.CountObservationsParams{
		StationIds:  arg.StationIds,
		IsStartDate: arg.IsStartDate,
		StartDate:   arg.StartDate,
		IsEndDate:   arg.IsEndDate,
		EndDate:     arg.EndDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList(req.Page, req.PerPage, int32(count), items)

	ctx.JSON(http.StatusOK, res)
}

type latestObsRes struct {
	Rain          util.Float4        `json:"rain"`
	Temp          util.Float4        `json:"temp"`
	Rh            util.Float4        `json:"rh"`
	Wdir          util.Float4        `json:"wdir"`
	Wspd          util.Float4        `json:"wspd"`
	Srad          util.Float4        `json:"srad"`
	Mslp          util.Float4        `json:"mslp"`
	Tn            util.Float4        `json:"tn"`
	Tx            util.Float4        `json:"tx"`
	Gust          util.Float4        `json:"gust"`
	RainAccum     util.Float4        `json:"rain_accum"`
	TnTimestamp   pgtype.Timestamptz `json:"tn_timestamp"`
	TxTimestamp   pgtype.Timestamptz `json:"tx_timestamp"`
	GustTimestamp pgtype.Timestamptz `json:"gust_timestamp"`
	Timestamp     pgtype.Timestamptz `json:"timestamp"`
}
type latestObservationRes struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Lat       util.Float4  `json:"lat"`
	Lon       util.Float4  `json:"lon"`
	Elevation util.Float4  `json:"elevation"`
	Address   pgtype.Text  `json:"address"`
	Obs       latestObsRes `json:"obs"`
} //@name LatestObservation

func newLatestObservationResponse(data any) latestObservationRes {
	switch d := data.(type) {
	case db.ListLatestObservationsRow:
		return latestObservationRes{
			ID:        d.ID,
			Name:      d.Name,
			Lat:       util.Float4{Float4: d.Lat},
			Lon:       util.Float4{Float4: d.Lon},
			Elevation: util.Float4{Float4: d.Elevation},
			Address:   d.Address,
			Obs: latestObsRes{
				Rain:          util.Float4{Float4: d.Rain},
				Temp:          util.Float4{Float4: d.Temp},
				Rh:            util.Float4{Float4: d.Rh},
				Wdir:          util.Float4{Float4: d.Wdir},
				Wspd:          util.Float4{Float4: d.Wspd},
				Srad:          util.Float4{Float4: d.Srad},
				Mslp:          util.Float4{Float4: d.Mslp},
				Tn:            util.Float4{Float4: d.Tn},
				Tx:            util.Float4{Float4: d.Tx},
				Gust:          util.Float4{Float4: d.Gust},
				RainAccum:     util.Float4{Float4: d.RainAccum},
				TnTimestamp:   d.TnTimestamp,
				TxTimestamp:   d.TxTimestamp,
				GustTimestamp: d.GustTimestamp,
				Timestamp:     d.Timestamp,
			},
		}
	case db.GetLatestStationObservationRow:
		return latestObservationRes{
			ID:        d.ID,
			Name:      d.Name,
			Lat:       util.Float4{Float4: d.Lat},
			Lon:       util.Float4{Float4: d.Lon},
			Elevation: util.Float4{Float4: d.Elevation},
			Address:   d.Address,
			Obs: latestObsRes{
				Rain:          util.Float4{Float4: d.ObservationsCurrent.Rain},
				Temp:          util.Float4{Float4: d.ObservationsCurrent.Temp},
				Rh:            util.Float4{Float4: d.ObservationsCurrent.Rh},
				Wdir:          util.Float4{Float4: d.ObservationsCurrent.Wdir},
				Wspd:          util.Float4{Float4: d.ObservationsCurrent.Wspd},
				Srad:          util.Float4{Float4: d.ObservationsCurrent.Srad},
				Mslp:          util.Float4{Float4: d.ObservationsCurrent.Mslp},
				Tn:            util.Float4{Float4: d.ObservationsCurrent.Tn},
				Tx:            util.Float4{Float4: d.ObservationsCurrent.Tx},
				Gust:          util.Float4{Float4: d.ObservationsCurrent.Gust},
				RainAccum:     util.Float4{Float4: d.ObservationsCurrent.RainAccum},
				TnTimestamp:   d.ObservationsCurrent.TnTimestamp,
				TxTimestamp:   d.ObservationsCurrent.TxTimestamp,
				GustTimestamp: d.ObservationsCurrent.GustTimestamp,
				Timestamp:     d.ObservationsCurrent.Timestamp,
			},
		}
	default:
		return latestObservationRes{}
	}
}

// ListLatestObservations
//
//	@Summary	list latest observation
//	@Tags		observations
//	@Produce	json
//	@Success	200	{array}	latestObservationRes
//	@Router		/observations/latest [get]
func (s *Server) ListLatestObservations(ctx *gin.Context) {
	_obsSlice, err := s.store.ListLatestObservations(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	obsSlice := make([]latestObservationRes, len(_obsSlice))

	for i := range obsSlice {
		obsSlice[i] = newLatestObservationResponse(_obsSlice[i])
	}

	ctx.JSON(http.StatusOK, obsSlice)
}

type getLatestStationObsReq struct {
	StationID int64 `uri:"station_id" binding:"required,min=1"`
}

// GetLatestStationObservation
//
//	@Summary	Get latest station observation
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path		int	true	"Station ID"
//	@Success	200			{object}	latestObservationRes
//	@Router		/stations/{station_id}/observations/latest [get]
func (s *Server) GetLatestStationObservation(ctx *gin.Context) {
	var req getLatestStationObsReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	obs, err := s.store.GetLatestStationObservation(ctx, req.StationID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station observation not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newLatestObservationResponse(obs))
}

type getNearestLatestStationObsReq struct {
	Pt string `form:"pt" binding:"required"`
} //@name GetNearestLatestStationObservationParams

// GetNearestLatestStationObservation
//
//	@Summary	Get nearest latest station observation
//	@Tags		observations
//	@Accept		json
//	@Produce	json
//	@Param		req	query		getNearestLatestStationObsReq	false	"Get nearest latest station observation parameters"
//	@Success	200	{object}	latestObservationRes
//	@Router		/stations/nearest/observations/latest [get]
func (s *Server) GetNearestLatestStationObservation(ctx *gin.Context) {
	var req getNearestLatestStationObsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ptArgs := strings.Split(req.Pt, ",")
	if len(ptArgs) != 2 {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid parameter: pt = %s", req.Pt)))
		return
	}
	lon, err := strconv.ParseFloat(ptArgs[0], 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err)
		return
	}
	lat, err := strconv.ParseFloat(ptArgs[1], 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err)
		return
	}

	obs, err := s.store.GetNearestLatestStationObservation(ctx, db.GetNearestLatestStationObservationParams{
		Lon: float32(lon),
		Lat: float32(lat),
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station observation not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newLatestObservationResponse(db.GetLatestStationObservationRow(obs)))
}
