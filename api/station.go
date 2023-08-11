package api

import (
	"errors"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type stationResponse struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Lat           util.NullFloat4 `json:"lat"`
	Lon           util.NullFloat4 `json:"lon"`
	Elevation     util.NullFloat4 `json:"elevation"`
	DateInstalled pgtype.Date     `json:"date_installed"`
	MobileNumber  util.NullString `json:"mobile_number"`
	StationType   util.NullString `json:"station_type"`
	StationType2  util.NullString `json:"station_type2"`
	StationUrl    util.NullString `json:"station_url"`
	Status        util.NullString `json:"status"`
	Province      util.NullString `json:"province"`
	Region        util.NullString `json:"region"`
	Address       util.NullString `json:"address"`
} //@name StationResponse

func newStationResponse(station db.ObservationsStation) stationResponse {
	return stationResponse{
		ID:            station.ID,
		Name:          station.Name,
		Lat:           station.Lat,
		Lon:           station.Lon,
		Elevation:     station.Elevation,
		DateInstalled: station.DateInstalled,
		MobileNumber:  station.MobileNumber,
		StationType:   station.StationType,
		StationType2:  station.StationType2,
		StationUrl:    station.StationUrl,
		Status:        station.Status,
		Province:      station.Province,
		Region:        station.Region,
		Address:       station.Address,
	}
}

type listStationsReq struct {
	Page    int32 `form:"page,default=1" binding:"omitempty,min=1"`            // page number
	PerPage int32 `form:"per_page,default=5" binding:"omitempty,min=1,max=30"` // limit
} //@name ListStationsParams

type listStationsRes struct {
	Page    int32             `json:"page"`
	PerPage int32             `json:"per_page"`
	Total   int64             `json:"total"`
	Data    []stationResponse `json:"data"`
} //@name ListStationsResponse

// ListStations
//
//	@Summary	List stations
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		req	query		listStationsReq	false	"List stations parameters"
//	@Success	200	{object}	listStationsRes
//	@Router		/stations [get]
func (s *Server) ListStations(ctx *gin.Context) {
	var req listStationsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListStationsParams{
		Limit:  req.PerPage,
		Offset: offset,
	}

	stations, err := s.store.ListStations(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numStations := len(stations)
	stationsRes := make([]stationResponse, numStations)
	for i, station := range stations {
		stationsRes[i] = newStationResponse(station)
	}

	totalStations, err := s.store.CountStations(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := listStationsRes{
		Page:    req.Page,
		PerPage: req.PerPage,
		Total:   totalStations,
		Data:    stationsRes,
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getStationReq struct {
	ID int64 `uri:"station_id" binding:"required,min=1"` // station id
}

// GetStation
//
//	@Summary	Get station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path		int	true	"Station ID"
//	@Success	200			{object}	stationResponse
//	@Router		/stations/{station_id} [get]
func (s *Server) GetStation(ctx *gin.Context) {
	var req getStationReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	station, err := s.store.GetStation(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newStationResponse(station))
}

type createStationReq struct {
	Name          string          `json:"name" binding:"required,alphanum"`
	Lat           util.NullFloat4 `json:"lat" binding:"omitempty,numeric"`
	Lon           util.NullFloat4 `json:"lon" binding:"omitempty,numeric"`
	Elevation     util.NullFloat4 `json:"elevation" binding:"omitempty,numeric"`
	DateInstalled pgtype.Date     `json:"date_installed"`
	MobileNumber  util.NullString `json:"mobile_number" binding:"omitempty"`
	StationType   util.NullString `json:"station_type" binding:"omitempty"`
	StationType2  util.NullString `json:"station_type2" binding:"omitempty"`
	StationUrl    util.NullString `json:"station_url" binding:"omitempty"`
	Status        util.NullString `json:"status" binding:"omitempty"`
	Province      util.NullString `json:"province" binding:"omitempty"`
	Region        util.NullString `json:"region" binding:"omitempty"`
	Address       util.NullString `json:"address" binding:"omitempty"`
} //@name CreateStationParams

// CreateStation
//
//	@Summary	Create station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		req	body	createStationReq	true	"Create station parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	stationResponse
//	@Router		/stations [post]
func (s *Server) CreateStation(ctx *gin.Context) {
	var req createStationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateStationParams{
		Name:          req.Name,
		Lat:           req.Lat,
		Lon:           req.Lon,
		Elevation:     req.Elevation,
		DateInstalled: req.DateInstalled,
		MobileNumber:  req.MobileNumber,
		StationType:   req.StationType,
		StationType2:  req.StationType2,
		StationUrl:    req.StationUrl,
		Status:        req.Status,
		Province:      req.Province,
		Region:        req.Region,
		Address:       req.Address,
	}

	result, err := s.store.CreateStation(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newStationResponse(result))
}

type updateStationUri struct {
	ID int64 `uri:"station_id" binding:"required,min=1"`
}

type updateStationReq struct {
	Name          util.NullString `json:"name" binding:"omitempty,alphanum"`
	Lat           util.NullFloat4 `json:"lat" binding:"omitempty,numeric"`
	Lon           util.NullFloat4 `json:"lon" binding:"omitempty,numeric"`
	Elevation     util.NullFloat4 `json:"elevation" binding:"omitempty,numeric"`
	DateInstalled pgtype.Date     `json:"date_installed"`
	MobileNumber  util.NullString `json:"mobile_number" binding:"omitempty"`
	StationType   util.NullString `json:"station_type" binding:"omitempty"`
	StationType2  util.NullString `json:"station_type2" binding:"omitempty"`
	StationUrl    util.NullString `json:"station_url" binding:"omitempty"`
	Status        util.NullString `json:"status" binding:"omitempty"`
	Province      util.NullString `json:"province" binding:"omitempty"`
	Region        util.NullString `json:"region" binding:"omitempty"`
	Address       util.NullString `json:"address" binding:"omitempty"`
} //@name UpdateStationParams

// UpdateStation
//
//	@Summary	Update station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int					true	"Station ID"
//	@Param		req			body	updateStationReq	true	"Update station parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	stationResponse
//	@Router		/stations/{station_id} [put]
func (s *Server) UpdateStation(ctx *gin.Context) {
	var uri updateStationUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateStationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.UpdateStationParams{
		ID:           uri.ID,
		Name:         req.Name,
		Lat:          req.Lat,
		Lon:          req.Lon,
		Elevation:    req.Elevation,
		MobileNumber: req.MobileNumber,
		StationType:  req.StationType,
		StationType2: req.StationType2,
		StationUrl:   req.StationUrl,
		Status:       req.Status,
		Province:     req.Province,
		Region:       req.Region,
		Address:      req.Address,
	}

	station, err := s.store.UpdateStation(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newStationResponse(station))
}

type deleteStationReq struct {
	ID int64 `uri:"station_id" binding:"required,min=1"`
}

// DeleteStation
//
//	@Summary	Delete station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int	true	"Station ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/stations/{station_id} [delete]
func (s *Server) DeleteStation(ctx *gin.Context) {
	var req deleteStationReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := s.store.DeleteStation(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
