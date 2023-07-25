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
} //@name Station

func newStationResponse(station db.ObservationsStation) stationResponse {
	return stationResponse{
		ID:           station.ID,
		Name:         station.Name,
		Lat:          station.Lat,
		Lon:          station.Lon,
		Elevation:    station.Elevation,
		MobileNumber: station.MobileNumber,
		StationType:  station.StationType,
		StationType2: station.StationType2,
		StationUrl:   station.StationUrl,
		Status:       station.Status,
		Province:     station.Province,
		Region:       station.Region,
		Address:      station.Address,
	}
}

type listStationReq struct {
	Page  int32 `form:"page,default=1" binding:"omitempty,min=1"`
	Limit int32 `form:"limit,default=5" binding:"omitempty,min=1,max=30"`
}

// ListStations godoc
// @Summary      List stations
// @Tags         stations
// @Produce      json
// @Success      200 {array} stationResponse
// @Router       /stations [get]
func (s *Server) ListStations(ctx *gin.Context) {
	var req listStationReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.Limit
	arg := db.ListStationsParams{
		Limit:  req.Limit,
		Offset: offset,
	}

	stations, err := s.store.ListStations(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numStations := len(stations)
	if numStations <= 0 {
		ctx.JSON(http.StatusOK, nil)
	}

	rsp := make([]stationResponse, numStations)
	for i, station := range stations {
		rsp[i] = newStationResponse(station)
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getStationReq struct {
	ID int64 `uri:"station_id" binding:"required,min=1"`
}

// GetStation godoc
// @Summary      Get station
// @Tags         stations
// @Produce      json
// @Success      200 {object} stationResponse
// @Router       /stations/{station_id} [get]
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
	MobileNumber  util.NullString `json:"mobile_number" binding:"omitempty,min=1"`
	StationType   util.NullString `json:"station_type" binding:"omitempty,min=1"`
	StationType2  util.NullString `json:"station_type2" binding:"omitempty,min=1"`
	StationUrl    util.NullString `json:"station_url" binding:"omitempty,min=1"`
	Status        util.NullString `json:"status" binding:"omitempty,min=1"`
	Province      util.NullString `json:"province" binding:"omitempty,min=1"`
	Region        util.NullString `json:"region" binding:"omitempty,min=1"`
	Address       util.NullString `json:"address" binding:"omitempty,min=1"`
}

// CreateStation godoc
// @Summary      Create station
// @Tags         stations
// @Produce      json
// @Success      201 {object} stationResponse
// @Router       /stations [post]
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
	MobileNumber  util.NullString `json:"mobile_number" binding:"omitempty,min=1"`
	StationType   util.NullString `json:"station_type" binding:"omitempty,min=1"`
	StationType2  util.NullString `json:"station_type2" binding:"omitempty,min=1"`
	StationUrl    util.NullString `json:"station_url" binding:"omitempty,min=1"`
	Status        util.NullString `json:"status" binding:"omitempty,min=1"`
	Province      util.NullString `json:"province" binding:"omitempty,min=1"`
	Region        util.NullString `json:"region" binding:"omitempty,min=1"`
	Address       util.NullString `json:"address" binding:"omitempty,min=1"`
}

// UpdateStation godoc
// @Summary      Update station
// @Tags         stations
// @Produce      json
// @Success      200 {object} stationResponse
// @Router       /stations/{station_id} [put]
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

// DeleteStation godoc
// @Summary      Delete station
// @Tags         stations
// @Produce      json
// @Success      204
// @Router       /stations/{station_id} [delete]
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
