package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateStation
//
//	@Summary	Create station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		req	body	models.CreateStationReq	true	"Create station parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	models.Station
//	@Router		/stations [post]
func (h *DefaultHandler) CreateStation(ctx *gin.Context) {
	var req models.CreateStationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := req.Transform()

	result, err := h.store.CreateStation(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := models.NewStation(result, false)
	ctx.JSON(http.StatusCreated, res)
}

type listStationsReq struct {
	Circle  string `form:"circle" binding:"omitempty"`
	BBox    string `form:"bbox" binding:"omitempty"`
	Status  string `form:"status" binding:"omitempty"`
	Page    int32  `form:"page,default=1" binding:"omitempty,min=1"` // page number
	PerPage int32  `form:"per_page" binding:"omitempty,min=1"`       // limit
} //@name ListStationsParams

type paginatedStations = util.PaginatedList[models.Station] //@name PaginatedStations

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
func (h *DefaultHandler) ListStations(ctx *gin.Context) {
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
		stations, err = h.store.ListStationsWithinRadius(
			ctx,
			db.ListStationsWithinRadiusParams{
				Cx:     float32(cX),
				Cy:     float32(cY),
				R:      float32(cR),
				Status: util.ToPgText(req.Status),
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
		stations, err = h.store.ListStationsWithinBBox(
			ctx,
			db.ListStationsWithinBBoxParams{
				Xmin:   float32(xMin),
				Ymin:   float32(yMin),
				Xmax:   float32(xMax),
				Ymax:   float32(yMax),
				Status: util.ToPgText(req.Status),
				Limit:  pgtype.Int4{Int32: req.PerPage, Valid: req.PerPage > 0},
				Offset: offset,
			})
	} else {
		arg := db.ListStationsParams{
			Status: util.ToPgText(req.Status),
			Limit:  pgtype.Int4{Int32: req.PerPage, Valid: req.PerPage > 0},
			Offset: offset,
		}
		stations, err = h.store.ListStations(ctx, arg)
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	key, exists := ctx.Get(models.AuthPayloadKey)
	var isSimpleResponse bool
	if exists {
		authPayload, ok := key.(*token.Payload)
		isSimpleResponse = !ok || authPayload == nil || len(authPayload.User.Username) == 0
	}

	numStations := len(stations)
	items := make([]models.Station, numStations)
	for i, station := range stations {
		items[i] = models.NewStation(station, isSimpleResponse)
	}

	var count int64
	if len(req.Circle) > 0 {
		count, err = h.store.CountStationsWithinRadius(
			ctx,
			db.CountStationsWithinRadiusParams{
				Cx:     float32(cX),
				Cy:     float32(cY),
				R:      float32(cR),
				Status: util.ToPgText(req.Status),
			})
	} else if len(req.BBox) > 0 {
		count, err = h.store.CountStationsWithinBBox(
			ctx,
			db.CountStationsWithinBBoxParams{
				Xmin:   float32(xMin),
				Ymin:   float32(yMin),
				Xmax:   float32(xMax),
				Ymax:   float32(yMax),
				Status: util.ToPgText(req.Status),
			})
	} else {
		count, err = h.store.CountStations(ctx, util.ToPgText(req.Status))
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList(req.Page, req.PerPage, int32(count), items)

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
//	@Success	200			{object}	models.Station
//	@Router		/stations/{station_id} [get]
func (h *DefaultHandler) GetStation(ctx *gin.Context) {
	var req getStationReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	station, err := h.store.GetStation(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	key, exists := ctx.Get(models.AuthPayloadKey)
	var isSimpleResponse bool
	if exists {
		authPayload, ok := key.(*token.Payload)
		isSimpleResponse = !ok || authPayload == nil || len(authPayload.User.Username) == 0
	}

	ctx.JSON(http.StatusOK, models.NewStation(station, isSimpleResponse))
}

type updateStationUri struct {
	ID int64 `uri:"station_id" binding:"required,min=1"`
}

// UpdateStation
//
//	@Summary	Update station
//	@Tags		stations
//	@Accept		json
//	@Produce	json
//	@Param		station_id	path	int						true	"Station ID"
//	@Param		req			body	models.UpdateStationReq	true	"Update station parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	models.Station
//	@Router		/stations/{station_id} [put]
func (h *DefaultHandler) UpdateStation(ctx *gin.Context) {
	var uri updateStationUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.UpdateStationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	req.ID = uri.ID

	arg := req.Transform()

	station, err := h.store.UpdateStation(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("station not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := models.NewStation(station, false)
	ctx.JSON(http.StatusOK, res)
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
func (h *DefaultHandler) DeleteStation(ctx *gin.Context) {
	var req deleteStationReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := h.store.DeleteStation(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
