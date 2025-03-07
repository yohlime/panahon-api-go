package sensor

import (
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type davisRawCurrentResponseV1 struct {
	Location   string                       `json:"location"`
	Lat        *float32                     `json:"latitude"`
	Lon        *float32                     `json:"longitude"`
	Time       string                       `json:"observation_time_rfc822"`
	PressureMb *float32                     `json:"pressure_mb"`
	Rh         *float32                     `json:"relative_humidity"`
	TempC      *float32                     `json:"temp_c"`
	TdC        *float32                     `json:"dewpoint_c"`
	WindDeg    *float32                     `json:"wind_degrees"`
	WindMPH    *float32                     `json:"wind_mph"`
	HeatIndexC *float32                     `json:"heat_index_c"`
	Obs        davisRawCurrentObservationV1 `json:"davis_current_observation"`
}

type davisRawCurrentObservationV1 struct {
	RRInPerHr       *float32 `json:"rain_rate_in_per_hr"`
	RainDayIn       *float32 `json:"rain_day_in"`
	Srad            *float32 `json:"solar_radiation"`
	UVIndex         *float32 `json:"uv_index"`
	TempDayHighF    *float32 `json:"temp_day_high_f"`
	TempDayLowF     *float32 `json:"temp_day_low_f"`
	WindDayHighMPH  *float32 `json:"wind_day_high_mph"`
	TempDayHighTime string   `json:"temp_day_high_time"`
	TempDayLowTime  string   `json:"temp_day_low_time"`
	WindDayHighTime string   `json:"wind_day_high_time"`
}

func (r davisRawCurrentResponseV1) ToDavisCurrentObservation() *DavisCurrentObservation {
	obs := DavisCurrentObservation{
		Rr:            util.ToFloat4(r.Obs.RRInPerHr),
		RainAccum:     util.ToFloat4(r.Obs.RainDayIn),
		Temp:          util.ToFloat4(r.TempC),
		Rh:            util.ToFloat4(r.Rh),
		Wdir:          util.ToFloat4(r.WindDeg),
		Wspd:          util.ToFloat4(r.WindMPH),
		Srad:          util.ToFloat4(r.Obs.Srad),
		Pres:          util.ToFloat4(r.PressureMb),
		Tx:            util.ToFloat4(r.Obs.TempDayHighF),
		Tn:            util.ToFloat4(r.Obs.TempDayLowF),
		Wspdx:         util.ToFloat4(r.Obs.WindDayHighMPH),
		Hi:            util.ToFloat4(r.HeatIndexC),
		TxTimestamp:   pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		TnTimestamp:   pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		GustTimestamp: pgtype.Timestamptz{Time: time.Time{}, Valid: true},
	}

	if obs.Rr.Valid {
		obs.Rr.Float32 = obs.Rr.Float32 * 25.4
	}
	if obs.RainAccum.Valid {
		obs.RainAccum.Float32 = obs.Rr.Float32 * 25.4
	}
	if obs.Wspd.Valid {
		obs.Wspd.Float32 = obs.Wspd.Float32 * 0.44704
	}
	if obs.Tx.Valid {
		obs.Tx.Float32 = util.FahrenheitToCelsius(obs.Tx.Float32)
		if dt, err := parseTimeStrToDateTime(r.Obs.TempDayHighTime); err == nil {
			obs.TxTimestamp.Time = dt
		}
	}
	if obs.Tn.Valid {
		obs.Tn.Float32 = util.FahrenheitToCelsius(obs.Tn.Float32)
		if dt, err := parseTimeStrToDateTime(r.Obs.TempDayLowTime); err == nil {
			obs.TnTimestamp.Time = dt
		}
	}
	if obs.Wspdx.Valid {
		obs.Wspdx.Float32 = obs.Wspdx.Float32 * 0.44704
		if dt, err := parseTimeStrToDateTime(r.Obs.WindDayHighTime); err == nil {
			obs.GustTimestamp.Time = dt
		}
	}

	layout := "Mon, 02 Jan 2006 15:04:05 -0700"
	if dt, err := time.Parse(layout, r.Time); err == nil {
		obs.Timestamp = pgtype.Timestamptz{Time: dt, Valid: true}
	}

	return &obs
}
