package sensor

import (
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type davisRawStationResponseV2 struct {
	StationID         int     `json:"station_id"`
	StationIDUUID     string  `json:"station_id_uuid"`
	StationName       string  `json:"station_name"`
	GatewayID         int     `json:"gateway_id"`
	GatewayIDHex      string  `json:"gateway_id_hex"`
	ProductNumber     string  `json:"product_number"`
	Username          string  `json:"username"`
	UserEmail         string  `json:"user_email"`
	CompanyName       string  `json:"company_name"`
	Active            bool    `json:"active"`
	Private           bool    `json:"private"`
	RecordingInterval int     `json:"recording_interval"`
	FirmwareVersion   string  `json:"firmware_version"`
	RegisteredDate    int64   `json:"registered_date"`
	TimeZone          string  `json:"time_zone"`
	City              string  `json:"city"`
	Region            string  `json:"region"`
	Country           string  `json:"country"`
	Latitude          float32 `json:"latitude"`
	Longitude         float32 `json:"longitude"`
	Elevation         float32 `json:"elevation"`
	GatewayType       string  `json:"gateway_type"`
	RelationshipType  string  `json:"relationship_type"`
	SubscriptionType  string  `json:"subscription_type"`
}

type davisRawStationsResponseV2 struct {
	Stations    []davisRawStationResponseV2 `json:"stations"`
	GeneratedAt int64                       `json:"generated_at"`
}

type davisRawCurrentResponseV2 struct {
	StationID     int                               `json:"station_id"`
	StationIDUUID string                            `json:"station_id_uuid"`
	Sensors       []davisRawCurrentSensorResponseV2 `json:"sensors"`
	GeneratedAt   int                               `json:"generated_at"`
}

type davisRawCurrentSensorResponseV2 struct {
	LSID              int                             `json:"lsid"`
	SensorType        int                             `json:"sensor_type"`
	DataStructureType int                             `json:"data_structure_type"`
	Data              []davisRawCurrentDataResponseV2 `json:"data"`
}

type davisRawCurrentDataResponseV2 struct {
	TS                 int64       `json:"ts"`
	TZOffset           int         `json:"tz_offset"`
	BarTrend           *float32    `json:"bar_trend"`
	Bar                *float32    `json:"bar"`
	TempIn             *float32    `json:"temp_in"`
	HumIn              *float32    `json:"hum_in"`
	TempOut            *float32    `json:"temp_out"`
	WindSpeed          *float32    `json:"wind_speed"`
	WindSpeed10MinAvg  *float32    `json:"wind_speed_10_min_avg"`
	WindDir            *float32    `json:"wind_dir"`
	TempExtra          [7]*float32 `json:"temp_extra"`
	TempSoil           [4]*float32 `json:"temp_soil"`
	TempLeaf           [4]*float32 `json:"temp_leaf"`
	HumOut             *float32    `json:"hum_out"`
	HumExtra           [7]*float32 `json:"hum_extra"`
	RainRateClicks     int         `json:"rain_rate_clicks"`
	RainRateIn         *float32    `json:"rain_rate_in"`
	RainRateMM         *float32    `json:"rain_rate_mm"`
	UV                 *float32    `json:"uv"`
	SolarRad           *float32    `json:"solar_rad"`
	RainStormClicks    int         `json:"rain_storm_clicks"`
	RainStormIn        *float32    `json:"rain_storm_in"`
	RainStormMM        *float32    `json:"rain_storm_mm"`
	RainStormStartDate *int64      `json:"rain_storm_start_date"`
	RainDayClicks      int         `json:"rain_day_clicks"`
	RainDayIn          *float32    `json:"rain_day_in"`
	RainDayMM          *float32    `json:"rain_day_mm"`
	RainMonthClicks    int         `json:"rain_month_clicks"`
	RainMonthIn        *float32    `json:"rain_month_in"`
	RainMonthMM        *float32    `json:"rain_month_mm"`
	RainYearClicks     int         `json:"rain_year_clicks"`
	RainYearIn         *float32    `json:"rain_year_in"`
	RainYearMM         *float32    `json:"rain_year_mm"`
	ETDay              *float32    `json:"et_day"`
	ETMonth            *float32    `json:"et_month"`
	ETYear             *float32    `json:"et_year"`
	MoistSoil          [4]*float32 `json:"moist_soil"`
	WetLeaf            [4]*float32 `json:"wet_leaf"`
	ForecastRule       int         `json:"forecast_rule"`
	ForecastDesc       string      `json:"forecast_desc"`
	DewPoint           *float32    `json:"dew_point"`
	HeatIndex          *float32    `json:"heat_index"`
	WindChill          *float32    `json:"wind_chill"`
	WindGust10Min      *float32    `json:"wind_gust_10_min"`
}

func (r davisRawCurrentResponseV2) ToDavisCurrentObservation() *DavisCurrentObservation {
	if len(r.Sensors) == 0 {
		return nil
	}
	rawSensor := r.Sensors[0]
	rawData := rawSensor.Data[0]
	obs := DavisCurrentObservation{
		Rr:        util.ToFloat4(rawData.RainRateMM),
		RainAccum: util.ToFloat4(rawData.RainDayMM),
		Temp:      util.ToFloat4(rawData.TempOut),
		Rh:        util.ToFloat4(rawData.HumOut),
		Wdir:      util.ToFloat4(rawData.WindDir),
		Wspd:      util.ToFloat4(rawData.WindSpeed),
		Srad:      util.ToFloat4(rawData.SolarRad),
		Pres:      util.ToFloat4(rawData.Bar),
		// Tx:        util.ToFloat4(rawObs.Obs.TempDayHighF),
		// Tn:        util.ToFloat4(rawObs.Obs.TempDayLowF),
		Wspdx:         util.ToFloat4(rawData.WindGust10Min),
		Hi:            util.ToFloat4(rawData.HeatIndex),
		TxTimestamp:   pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		TnTimestamp:   pgtype.Timestamptz{Time: time.Time{}, Valid: true},
		GustTimestamp: pgtype.Timestamptz{Time: time.Time{}, Valid: true},
	}

	if obs.Temp.Valid {
		obs.Temp.Float32 = util.FahrenheitToCelsius(obs.Temp.Float32)
	}
	if obs.Pres.Valid {
		obs.Pres.Float32 = util.InHgToMbar(obs.Pres.Float32)
	}
	if obs.Wspd.Valid {
		obs.Wspd.Float32 = obs.Wspd.Float32 * 0.44704
	}
	if obs.Wspdx.Valid {
		obs.Wspdx.Float32 = obs.Wspdx.Float32 * 0.44704
	}
	if obs.Hi.Valid {
		obs.Hi.Float32 = util.FahrenheitToCelsius(obs.Hi.Float32)
	}

	if rawData.TS > 0 {
		t := time.Unix(rawData.TS, 0) //.Add(time.Duration(rawData.TZOffset * int(time.Second)))
		obs.Timestamp = pgtype.Timestamptz{Time: t, Valid: true}
	}

	return &obs
}
