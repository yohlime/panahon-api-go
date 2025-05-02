package sensor

import (
	"fmt"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type davisRawWeatherDataResponseDashboard struct {
	OwnerName           *string                               `json:"ownerName"`
	LastReceived        int64                                 `json:"lastReceived"`
	CurrConditionValues []davisRawSensorDataResponseDashboard `json:"currConditionValues"`
	HighLowValues       []davisRawSensorDataResponseDashboard `json:"highLowValues"`
}

type davisRawSensorDataResponseDashboard struct {
	SensorDataTypeID      *int     `json:"sensorDataTypeId"`
	SensorDataName        string   `json:"sensorDataName"`
	DisplayName           *string  `json:"displayName"`
	ReportedValue         *float32 `json:"reportedValue"`
	Value                 *float32 `json:"value"`
	ConvertedValue        string   `json:"convertedValue"`
	DepthLabel            *string  `json:"depthLabel"`
	Category              *string  `json:"category"`
	AssocSensorDataTypeID *int     `json:"assocSensorDataTypeId"`
	SortOrder             *int     `json:"sortOrder"`
	UnitLabel             *string  `json:"unitLabel"`
}

func (r davisRawWeatherDataResponseDashboard) ToDavisCurrentObservation() *DavisCurrentObservation {
	obs := DavisCurrentObservation{
		TxTimestamp:   pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		TnTimestamp:   pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		GustTimestamp: pgtype.Timestamptz{Time: time.Time{}, Valid: true},
	}
	for _, cur := range r.CurrConditionValues {
		if cur.SensorDataTypeID == nil {
			continue
		}

		switch *cur.SensorDataTypeID {
		case 7:
			obs.Temp = util.ToFloat4(cur.Value)

		case 12:
			obs.Hi = util.ToFloat4(cur.Value)

		case 14:
			obs.Rh = util.ToFloat4(cur.Value)

		case 15:
			obs.Wspd = util.ToFloat4(cur.Value)

		case 17:
			obs.Wdir = util.ToFloat4(cur.Value)

		case 20:
			if cur.SensorDataName == "60 Min Rain Total" {
				obs.RainAccum = util.ToFloat4(cur.Value)
			}

		case 22:
			obs.Rr = util.ToFloat4(cur.Value)

		case 26:
			obs.Pres = util.ToFloat4(cur.Value)

		case 28:
			obs.Srad = util.ToFloat4(cur.Value)

		// case 56:
		// 	obs.Gust = util.ToFloat4(cur.Value)

		default:
			continue
		}
	}

	for _, hl := range r.HighLowValues {
		if hl.SensorDataTypeID != nil {
			switch *hl.SensorDataTypeID {
			case 57:
				obs.Tx = util.ToFloat4(hl.Value)

			case 58:
				obs.Tn = util.ToFloat4(hl.Value)

			case 65:
				obs.Wspdx = util.ToFloat4(hl.Value)

			default:
				continue
			}
		} else {
			switch hl.SensorDataName {
			case "High Temp Time":
				if dt, err := convertToTime(hl.Value); err == nil {
					obs.TxTimestamp.Time = dt
				}
			case "Low Temp Time":
				if dt, err := convertToTime(hl.Value); err == nil {
					obs.TnTimestamp.Time = dt
				}
			case "High Wind Speed Time":
				if dt, err := convertToTime(hl.Value); err == nil {
					obs.GustTimestamp.Time = dt
				}
			}
		}
	}

	if obs.Wspd.Valid {
		obs.Wspd.Float32 = obs.Wspd.Float32 * 0.44704
	}
	if obs.Wspdx.Valid {
		obs.Wspdx.Float32 = obs.Wspdx.Float32 * 0.44704
	}

	if r.LastReceived > 0 {
		dt := time.UnixMilli(r.LastReceived)
		obs.Timestamp = pgtype.Timestamptz{Time: dt, Valid: true}
	}

	return &obs
}

func convertToTime(hhmm *float32) (time.Time, error) {
	if hhmm == nil {
		return time.Time{}, fmt.Errorf("invalid time format")
	}

	hhmmInt := int(*hhmm)
	hours := hhmmInt / 100
	minutes := hhmmInt % 100

	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return time.Time{}, fmt.Errorf("invalid time format: %f", *hhmm)
	}

	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hours, minutes, 0, 0, time.Local), nil
}
