package models

import (
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type BaseStation struct {
	Lat           *float32      `json:"lat" fake:"{latitude}"`
	Lon           *float32      `json:"lon" fake:"{longitude}"`
	Elevation     *float32      `json:"elevation" fake:"{float32range:0,30}"`
	DateInstalled util.Date     `json:"date_installed,omitempty"`
	MobileNumber  string        `json:"mobile_number,omitempty"`
	StationType   string        `json:"station_type,omitempty"`
	StationType2  string        `json:"station_type2,omitempty"`
	StationUrl    string        `json:"station_url,omitempty"`
	Status        string        `json:"status,omitempty"`
	Province      util.Province `json:"province"`
	Region        util.Region   `json:"region"`
	Address       string        `json:"address"`
}

type Station struct {
	ID   int64  `json:"id"`
	Name string `json:"name" fake:"{lettern:12}"`
	BaseStation
} //@name Station

// NewStation creates new Station from db.ObservationsStation
func NewStation(station db.ObservationsStation, simple bool) Station {
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
			res.DateInstalled = util.Date{Time: station.DateInstalled.Time}
		}
	}
	if station.Province.Valid {
		res.Province = util.Province(station.Province.String)
	}
	if station.Region.Valid {
		res.Region = util.Region(station.Region.String)
	}
	if station.Address.Valid {
		res.Address = station.Address.String
	}

	return res
}

type CreateStationReq struct {
	Name string `json:"name" binding:"required"`
	BaseStation
} //@name CreateStationReq

func (r CreateStationReq) Transform() db.CreateStationParams {
	return transformStation(r.BaseStation, db.CreateStationParams{Name: r.Name})
}

type UpdateStationReq struct {
	ID   int64
	Name string `json:"name" binding:"omitempty"`
	BaseStation
} //@name UpdateStationReq

func (r UpdateStationReq) Transform() db.UpdateStationParams {
	return transformStation(
		r.BaseStation,
		db.UpdateStationParams{
			ID:   r.ID,
			Name: pgtype.Text{String: r.Name, Valid: len(r.Name) > 0},
		})
}

type StationParams interface {
	db.CreateStationParams | db.UpdateStationParams
}

func transformStation[T StationParams](req BaseStation, extraParams T) T {
	arg := db.CreateStationParams{
		Lat:       util.ToFloat4(req.Lat),
		Lon:       util.ToFloat4(req.Lon),
		Elevation: util.ToFloat4(req.Elevation),
		DateInstalled: pgtype.Date{
			Time:  req.DateInstalled.Time,
			Valid: !req.DateInstalled.IsZero(),
		},
		MobileNumber: util.ToPgText(req.MobileNumber),
		StationType:  util.ToPgText(req.StationType),
		StationType2: util.ToPgText(req.StationType2),
		StationUrl:   util.ToPgText(req.StationUrl),
		Status:       util.ToPgText(req.Status),
		Province:     util.ToPgText(string(req.Province)),
		Region:       util.ToPgText(string(req.Region)),
		Address:      util.ToPgText(req.Address),
	}

	switch v := any(extraParams).(type) {
	case db.CreateStationParams:
		arg.Name = v.Name
		return any(arg).(T)
	case db.UpdateStationParams:
		return any(db.UpdateStationParams{
			ID:            v.ID,
			Name:          v.Name,
			Lat:           arg.Lat,
			Lon:           arg.Lon,
			Elevation:     arg.Elevation,
			DateInstalled: arg.DateInstalled,
			MobileNumber:  arg.MobileNumber,
			StationType:   arg.StationType,
			StationType2:  arg.StationType2,
			StationUrl:    arg.StationUrl,
			Status:        arg.Status,
			Province:      arg.Province,
			Region:        arg.Region,
			Address:       arg.Address,
		}).(T)
	default:
		panic("Unsupported type")
	}
}
