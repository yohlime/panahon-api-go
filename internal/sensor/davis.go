package sensor

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
)

type Davis struct {
	Url    string
	client Fetcher
}

type DavisCurrentObservation struct {
	Rain          pgtype.Float4      `json:"rain"`
	Temp          pgtype.Float4      `json:"temp"`
	Rh            pgtype.Float4      `json:"rh"`
	Wdir          pgtype.Float4      `json:"wdir"`
	Wspd          pgtype.Float4      `json:"wspd"`
	Srad          pgtype.Float4      `json:"srad"`
	Mslp          pgtype.Float4      `json:"mslp"`
	Tn            pgtype.Float4      `json:"tn"`
	Tx            pgtype.Float4      `json:"tx"`
	Gust          pgtype.Float4      `json:"gust"`
	Hi            pgtype.Float4      `json:"hi"`
	RainAccum     pgtype.Float4      `json:"rain_accum"`
	TnTimestamp   pgtype.Timestamptz `json:"tn_timestamp"`
	TxTimestamp   pgtype.Timestamptz `json:"tx_timestamp"`
	GustTimestamp pgtype.Timestamptz `json:"gust_timestamp"`
	Timestamp     pgtype.Timestamptz `json:"timestamp"`
}

type davisRawResponse struct {
	Location   string                     `json:"location"`
	Lat        json.Number                `json:"latitude"`
	Lon        json.Number                `json:"longitude"`
	Time       string                     `json:"observation_time_rfc822"`
	PressureMb json.Number                `json:"pressure_mb"`
	Rh         json.Number                `json:"relative_humidity"`
	TempC      json.Number                `json:"temp_c"`
	TdC        json.Number                `json:"dewpoint_c"`
	WindDeg    json.Number                `json:"wind_degrees"`
	WindMPH    json.Number                `json:"wind_mph"`
	HeatIndexC json.Number                `json:"heat_index_c"`
	Obs        davisRawCurrentObservation `json:"davis_current_observation"`
}

type davisRawCurrentObservation struct {
	RRInPerHr       json.Number `json:"rain_rate_in_per_hr"`
	RainDayIn       json.Number `json:"rain_day_in"`
	Srad            json.Number `json:"solar_radiation"`
	UVIndex         json.Number `json:"uv_index"`
	TempDayHighF    json.Number `json:"temp_day_high_f"`
	TempDayLowF     json.Number `json:"temp_day_low_f"`
	WindDayHighMPH  json.Number `json:"wind_day_high_mph"`
	TempDayHighTime string      `json:"temp_day_high_time"`
	TempDayLowTime  string      `json:"temp_day_low_time"`
	WindDayHighTime string      `json:"wind_day_high_time"`
}

func NewDavis(url string) *Davis {
	client := &http.Client{Timeout: 10 * time.Second}
	return &Davis{
		Url:    url,
		client: client,
	}
}

func newDavisObservation(rawObs davisRawResponse) *DavisCurrentObservation {
	obs := new(DavisCurrentObservation)
	f, err := rawObs.Obs.RRInPerHr.Float64()
	obs.Rain = pgtype.Float4{Float32: float32(f) * 0.2, Valid: err == nil}
	f, err = rawObs.Obs.RainDayIn.Float64()
	obs.RainAccum = pgtype.Float4{Float32: float32(f) * 0.2, Valid: err == nil}
	f, err = rawObs.TempC.Float64()
	obs.Temp = pgtype.Float4{Float32: float32(f), Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.Rh.Float64()
	obs.Rh = pgtype.Float4{Float32: float32(f), Valid: err == nil && f >= 0 && f <= 100}
	f, err = rawObs.WindDeg.Float64()
	obs.Wdir = pgtype.Float4{Float32: float32(f), Valid: err == nil && f >= 0.0 && f <= 360.0}
	f, err = rawObs.WindMPH.Float64()
	obs.Wspd = pgtype.Float4{Float32: float32(f) * 0.44704, Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.Obs.Srad.Float64()
	obs.Srad = pgtype.Float4{Float32: float32(f), Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.PressureMb.Float64()
	obs.Mslp = pgtype.Float4{Float32: float32(f), Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.Obs.TempDayHighF.Float64()
	obs.Tx = pgtype.Float4{Float32: (float32(f) - 32.0) * (5.0 / 9.0), Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.Obs.TempDayLowF.Float64()
	obs.Tn = pgtype.Float4{Float32: (float32(f) - 32.0) * (5.0 / 9.0), Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.Obs.WindDayHighMPH.Float64()
	obs.Gust = pgtype.Float4{Float32: float32(f) * 0.44704, Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	f, err = rawObs.HeatIndexC.Float64()
	obs.Hi = pgtype.Float4{Float32: float32(f), Valid: err == nil && math.Abs(-999.0-f) > 0.001}
	dt, err := parseTimeStrToDateTime(rawObs.Obs.TempDayLowTime)
	if err == nil {
		obs.TnTimestamp = pgtype.Timestamptz{Time: dt, Valid: true}
	}
	dt, err = parseTimeStrToDateTime(rawObs.Obs.TempDayHighTime)
	if err == nil {
		obs.TxTimestamp = pgtype.Timestamptz{Time: dt, Valid: true}
	}
	dt, err = parseTimeStrToDateTime(rawObs.Obs.WindDayHighTime)
	if err == nil {
		obs.GustTimestamp = pgtype.Timestamptz{Time: dt, Valid: true}
	}

	layout := "Mon, 02 Jan 2006 15:04:05 -0700"
	t, err := time.Parse(layout, rawObs.Time)
	if err == nil {
		obs.Timestamp = pgtype.Timestamptz{Time: t, Valid: true}
	}
	return obs
}

func (d Davis) FetchLatest() (*DavisCurrentObservation, error) {
	req, err := http.NewRequest("GET", d.Url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36")

	sleepDuration := time.Duration(rand.Intn(4)+2) * time.Second
	time.Sleep(sleepDuration)

	res, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var rawObs davisRawResponse
	err = json.NewDecoder(res.Body).Decode(&rawObs)
	if err != nil {
		return nil, err
	}

	obs := newDavisObservation(rawObs)
	return obs, nil
}

func parseTimeStrToDateTime(timeStr string) (time.Time, error) {
	layout := "3:04pm"
	currentDate := time.Now()

	t, err := time.Parse(layout, timeStr)
	if err != nil {
		return currentDate, err
	}

	newDateTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), t.Hour(), t.Minute(), 0, 0, currentDate.Location())
	return newDateTime, nil
}

func RandomDavisRawResponse() davisRawResponse {
	return davisRawResponse{
		Location:   util.RandomString(24),
		Lat:        json.Number(fmt.Sprintf("%.2f", util.RandomFloat(4.0, 22.0))),
		Lon:        json.Number(fmt.Sprintf("%.2f", util.RandomFloat(114.0, 121.0))),
		PressureMb: json.Number(fmt.Sprintf("%.2f", util.RandomFloat(990.0, 1100.))),
		Rh:         json.Number(fmt.Sprintf("%.2f", util.RandomFloat(0.0, 100.0))),
		TempC:      json.Number(fmt.Sprintf("%.2f", util.RandomFloat(25.0, 33.0))),
		TdC:        json.Number(fmt.Sprintf("%.2f", util.RandomFloat(25.0, 33.0))),
		WindDeg:    json.Number(fmt.Sprintf("%d", int32(util.RandomInt(0, 360)))),
		WindMPH:    json.Number(fmt.Sprintf("%.2f", util.RandomFloat(0.0, 10.0))),
		HeatIndexC: json.Number(fmt.Sprintf("%.2f", util.RandomFloat(30.0, 50.0))),
		Obs: davisRawCurrentObservation{
			RRInPerHr:       json.Number(fmt.Sprintf("%.2f", util.RandomFloat(0.0, 5.0))),
			RainDayIn:       json.Number(fmt.Sprintf("%.2f", util.RandomFloat(0.0, 100.0))),
			Srad:            json.Number(fmt.Sprintf("%d", int32(util.RandomInt(0, 400)))),
			UVIndex:         json.Number(fmt.Sprintf("%.2f", util.RandomFloat(0.0, 1.0))),
			TempDayHighF:    json.Number(fmt.Sprintf("%.2f", util.RandomFloat(77.0, 104.0))),
			TempDayLowF:     json.Number(fmt.Sprintf("%.2f", util.RandomFloat(60.0, 104.0))),
			WindDayHighMPH:  json.Number(fmt.Sprintf("%.2f", util.RandomFloat(0.0, 20.0))),
			TempDayHighTime: randomTimeString(),
			TempDayLowTime:  randomTimeString(),
			WindDayHighTime: randomTimeString(),
		},
		Time: time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"),
	}
}

func randomTimeString() string {
	h := util.RandomInt(1, 12)
	m := util.RandomInt(0, 59)
	i := util.RandomInt(0, 100)
	x := "am"
	if i%2 == 0 {
		x = "pm"
	}

	return fmt.Sprintf("%d:%02d%s", h, m, x)
}
