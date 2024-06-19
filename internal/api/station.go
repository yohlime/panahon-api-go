package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Station struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Lat           *float32 `json:"lat"`
	Lon           *float32 `json:"lon"`
	Elevation     *float32 `json:"elevation"`
	DateInstalled string   `json:"date_installed,omitempty"`
	MobileNumber  string   `json:"mobile_number,omitempty"`
	StationType   string   `json:"station_type,omitempty"`
	StationType2  string   `json:"station_type2,omitempty"`
	StationUrl    string   `json:"station_url,omitempty"`
	Status        string   `json:"status,omitempty"`
	Province      string   `json:"province"`
	Region        string   `json:"region"`
	Address       string   `json:"address"`
} //@name Station

func newStation(station db.ObservationsStation, simple bool) Station {
	res := Station{
		ID:   station.ID,
		Name: station.Name,
	}

	if station.Lat.Valid {
		res.Lat = &station.Lat.Float32
	}
	if station.Lon.Valid {
		res.Lon = &station.Lon.Float32
	}
	if station.Elevation.Valid {
		res.Elevation = &station.Elevation.Float32
	}
	if !simple {
		if station.MobileNumber.Valid {
			res.MobileNumber = station.MobileNumber.String
		}
		if station.StationType.Valid {
			res.StationType = station.StationType.String
		}
		if station.StationType2.Valid {
			res.StationType2 = station.StationType2.String
		}
		if station.StationUrl.Valid {
			res.StationUrl = station.StationUrl.String
		}
		if station.Status.Valid {
			res.Status = station.Status.String
		}
		if station.Status.Valid {
			res.DateInstalled = station.DateInstalled.Time.Format("2006-01-02")
		}
	}
	if station.Province.Valid {
		res.Province = station.Province.String
	}
	if station.Region.Valid {
		res.Region = station.Region.String
	}
	if station.Address.Valid {
		res.Address = station.Address.String
	}

	return res
}

type createStationReq struct {
	Name          string   `json:"name" binding:"required,alphanumspace"`
	Lat           *float32 `json:"lat"`
	Lon           *float32 `json:"lon"`
	Elevation     *float32 `json:"elevation"`
	DateInstalled string   `json:"date_installed"`
	MobileNumber  string   `json:"mobile_number"`
	StationType   string   `json:"station_type"`
	StationType2  string   `json:"station_type2"`
	StationUrl    string   `json:"station_url"`
	Status        string   `json:"status"`
	Province      string   `json:"province"`
	Region        string   `json:"region"`
	Address       string   `json:"address"`
} //@name CreateStationParams

// CreateStation
//
//	@Summary	Create station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		req	body	createStationReq	true	"Create station parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	Station
//	@Router		/stations [post]
func (s *Server) CreateStation(ctx *gin.Context) {
	var req createStationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var dateInstalled pgtype.Date
	if len(req.DateInstalled) == 10 {
		_dt, err := time.Parse("2006-01-02", req.DateInstalled)
		dateInstalled = pgtype.Date{Time: _dt, Valid: err == nil}
	}

	arg := db.CreateStationParams{
		Name:          req.Name,
		DateInstalled: dateInstalled,
		MobileNumber: pgtype.Text{
			String: req.MobileNumber,
			Valid:  len(req.MobileNumber) > 0,
		},
		StationType: pgtype.Text{
			String: req.StationType,
			Valid:  len(req.StationType) > 0,
		},
		StationType2: pgtype.Text{
			String: req.StationType2,
			Valid:  len(req.StationType2) > 0,
		},
		StationUrl: pgtype.Text{
			String: req.StationUrl,
			Valid:  len(req.StationUrl) > 0,
		},
		Status: pgtype.Text{
			String: req.Status,
			Valid:  len(req.Status) > 0,
		},
		Province: pgtype.Text{
			String: req.Province,
			Valid:  len(req.Province) > 0,
		},
		Region: pgtype.Text{
			String: req.Region,
			Valid:  len(req.Region) > 0,
		},
		Address: pgtype.Text{
			String: req.Address,
			Valid:  len(req.Address) > 0,
		},
	}

	if req.Lat != nil {
		arg.Lat = pgtype.Float4{
			Float32: *req.Lat,
			Valid:   true,
		}
	}
	if req.Lon != nil {
		arg.Lon = pgtype.Float4{
			Float32: *req.Lon,
			Valid:   true,
		}
	}
	if req.Elevation != nil {
		arg.Elevation = pgtype.Float4{
			Float32: *req.Elevation,
			Valid:   true,
		}
	}

	result, err := s.store.CreateStation(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newStation(result, false))
}

type listStationsReq struct {
	Circle  string `form:"circle" binding:"omitempty"`
	BBox    string `form:"bbox" binding:"omitempty"`
	Status  string `form:"status" binding:"omitempty"`
	Page    int32  `form:"page,default=1" binding:"omitempty,min=1"` // page number
	PerPage int32  `form:"per_page" binding:"omitempty,min=1"`       // limit
} //@name ListStationsParams

type paginatedStations = util.PaginatedList[Station] //@name PaginatedStations

// ListStations
//
//	@Summary	List stations
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		req	query	listStationsReq	false	"List stations parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	paginatedStations
//	@Router		/stations [get]
func (s *Server) ListStations(ctx *gin.Context) {
	var req listStationsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage

	var stations []db.ObservationsStation
	var err error
	var cX, cY, cR float64
	var xMin, yMin, xMax, yMax float64
	if len(req.Circle) > 0 {
		cArgs := strings.Split(req.Circle, ",")
		if len(cArgs) != 3 {
			ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid parameter: circle = %s", req.Circle)))
			return
		}
		for i := range cArgs {
			switch i {
			case 0:
				cX, err = strconv.ParseFloat(cArgs[i], 32)
			case 1:
				cY, err = strconv.ParseFloat(cArgs[i], 32)
			case 2:
				cR, err = strconv.ParseFloat(cArgs[i], 32)
			default:
				continue
			}
			if err != nil {
				ctx.JSON(http.StatusBadRequest, err)
				return
			}
		}
		stations, err = s.store.ListStationsWithinRadius(
			ctx,
			db.ListStationsWithinRadiusParams{
				Cx:     float32(cX),
				Cy:     float32(cY),
				R:      float32(cR),
				Status: pgtype.Text{String: req.Status, Valid: len(req.Status) > 0},
				Limit:  pgtype.Int4{Int32: req.PerPage, Valid: req.PerPage > 0},
				Offset: offset,
			})
	} else if len(req.BBox) > 0 {
		rArgs := strings.Split(req.BBox, ",")
		if len(rArgs) != 4 {
			ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid parameter: bbox = %s", req.BBox)))
			return
		}
		for i := range rArgs {
			switch i {
			case 0:
				xMin, err = strconv.ParseFloat(rArgs[i], 32)
			case 1:
				yMin, err = strconv.ParseFloat(rArgs[i], 32)
			case 2:
				xMax, err = strconv.ParseFloat(rArgs[i], 32)
			case 3:
				yMax, err = strconv.ParseFloat(rArgs[i], 32)
			default:
				continue
			}
			if err != nil {
				ctx.JSON(http.StatusBadRequest, err)
				return
			}
		}
		stations, err = s.store.ListStationsWithinBBox(
			ctx,
			db.ListStationsWithinBBoxParams{
				Xmin:   float32(xMin),
				Ymin:   float32(yMin),
				Xmax:   float32(xMax),
				Ymax:   float32(yMax),
				Status: pgtype.Text{String: req.Status, Valid: len(req.Status) > 0},
				Limit:  pgtype.Int4{Int32: req.PerPage, Valid: req.PerPage > 0},
				Offset: offset,
			})
	} else {
		arg := db.ListStationsParams{
			Status: pgtype.Text{String: req.Status, Valid: len(req.Status) > 0},
			Limit:  pgtype.Int4{Int32: req.PerPage, Valid: req.PerPage > 0},
			Offset: offset,
		}
		stations, err = s.store.ListStations(ctx, arg)
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	key, exists := ctx.Get(authPayloadKey)
	var isSimpleResponse bool
	if exists {
		authPayload, ok := key.(*token.Payload)
		isSimpleResponse = !ok || authPayload == nil || len(authPayload.User.Username) == 0
	}

	numStations := len(stations)
	items := make([]Station, numStations)
	for i, station := range stations {
		items[i] = newStation(station, isSimpleResponse)
	}

	var count int64
	if len(req.Circle) > 0 {
		count, err = s.store.CountStationsWithinRadius(
			ctx,
			db.CountStationsWithinRadiusParams{
				Cx:     float32(cX),
				Cy:     float32(cY),
				R:      float32(cR),
				Status: pgtype.Text{String: req.Status, Valid: len(req.Status) > 0},
			})
	} else if len(req.BBox) > 0 {
		count, err = s.store.CountStationsWithinBBox(
			ctx,
			db.CountStationsWithinBBoxParams{
				Xmin:   float32(xMin),
				Ymin:   float32(yMin),
				Xmax:   float32(xMax),
				Ymax:   float32(yMax),
				Status: pgtype.Text{String: req.Status, Valid: len(req.Status) > 0},
			})
	} else {
		count, err = s.store.CountStations(ctx, pgtype.Text{String: req.Status, Valid: len(req.Status) > 0})
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList[Station](req.Page, req.PerPage, int32(count), items)

	ctx.JSON(http.StatusOK, res)
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
//	@Success	200			{object}	Station
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

	key, exists := ctx.Get(authPayloadKey)
	var isSimpleResponse bool
	if exists {
		authPayload, ok := key.(*token.Payload)
		isSimpleResponse = !ok || authPayload == nil || len(authPayload.User.Username) == 0
	}

	ctx.JSON(http.StatusOK, newStation(station, isSimpleResponse))
}

type updateStationUri struct {
	ID int64 `uri:"station_id" binding:"required,min=1"`
}

type updateStationReq struct {
	Name          string   `json:"name" binding:"omitempty,alphanumspace"`
	Lat           *float32 `json:"lat"`
	Lon           *float32 `json:"lon"`
	Elevation     *float32 `json:"elevation"`
	DateInstalled string   `json:"date_installed"`
	MobileNumber  string   `json:"mobile_number"`
	StationType   string   `json:"station_type"`
	StationType2  string   `json:"station_type2"`
	StationUrl    string   `json:"station_url"`
	Status        string   `json:"status"`
	Province      string   `json:"province"`
	Region        string   `json:"region"`
	Address       string   `json:"address"`
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
//	@Success	200	{object}	Station
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

	var dateInstalled pgtype.Date
	if len(req.DateInstalled) == 10 {
		_dt, err := time.Parse("2006-01-02", req.DateInstalled)
		dateInstalled = pgtype.Date{Time: _dt, Valid: err == nil}
	}

	arg := db.UpdateStationParams{
		ID: uri.ID,
		Name: pgtype.Text{
			String: req.Name,
			Valid:  len(req.Name) > 0,
		},
		DateInstalled: dateInstalled,
		MobileNumber: pgtype.Text{
			String: req.MobileNumber,
			Valid:  len(req.MobileNumber) > 0,
		},
		StationType: pgtype.Text{
			String: req.StationType,
			Valid:  len(req.StationType) > 0,
		},
		StationType2: pgtype.Text{
			String: req.StationType2,
			Valid:  len(req.StationType2) > 0,
		},
		StationUrl: pgtype.Text{
			String: req.StationUrl,
			Valid:  len(req.StationUrl) > 0,
		},
		Status: pgtype.Text{
			String: req.Status,
			Valid:  len(req.Status) > 0,
		},
		Province: pgtype.Text{
			String: req.Province,
			Valid:  len(req.Province) > 0,
		},
		Region: pgtype.Text{
			String: req.Region,
			Valid:  len(req.Region) > 0,
		},
		Address: pgtype.Text{
			String: req.Address,
			Valid:  len(req.Address) > 0,
		},
	}

	if req.Lat != nil {
		arg.Lat = pgtype.Float4{
			Float32: *req.Lat,
			Valid:   true,
		}
	}
	if req.Lon != nil {
		arg.Lon = pgtype.Float4{
			Float32: *req.Lon,
			Valid:   true,
		}
	}
	if req.Elevation != nil {
		arg.Elevation = pgtype.Float4{
			Float32: *req.Elevation,
			Valid:   true,
		}
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

	ctx.JSON(http.StatusOK, newStation(station, false))
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
