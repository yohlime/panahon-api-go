package models

import (
	"fmt"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type BaseStation struct {
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
}

type Station struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
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

func RandomStation() Station {
	lat := util.RandomFloat[float32](-90.0, 90.0)
	lon := util.RandomFloat[float32](0.0, 360.0)
	return Station{
		ID:   util.RandomInt[int64](1, 1000),
		Name: fmt.Sprintf("%s %s", util.RandomString(12), util.RandomString(8)),
		BaseStation: BaseStation{
			DateInstalled: fmt.Sprintf("%d-%02d-%02d", util.RandomInt(2000, 2023), util.RandomInt(1, 12), util.RandomInt(1, 25)),
			Lat:           &lat,
			Lon:           &lon,
			Province:      util.RandomString(16),
			Region:        util.RandomString(16),
		},
	}
}

type CreateStationReq struct {
	Name string `json:"name" binding:"required,alphanumspace"`
	BaseStation
} //@name CreateStationReq

func (r CreateStationReq) Transform() db.CreateStationParams {
	return transformStation(r.BaseStation, db.CreateStationParams{Name: r.Name})
}

type UpdateStationReq struct {
	ID   int64
	Name string `json:"name" binding:"omitempty,alphanumspace"`
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
		Lat:           util.ToFloat4(req.Lat),
		Lon:           util.ToFloat4(req.Lon),
		Elevation:     util.ToFloat4(req.Elevation),
		DateInstalled: util.ToPgDate(req.DateInstalled),
		MobileNumber:  util.ToPgText(req.MobileNumber),
		StationType:   util.ToPgText(req.StationType),
		StationType2:  util.ToPgText(req.StationType2),
		StationUrl:    util.ToPgText(req.StationUrl),
		Status:        util.ToPgText(req.Status),
		Province:      util.ToPgText(req.Province),
		Region:        util.ToPgText(req.Region),
		Address:       util.ToPgText(req.Address),
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
