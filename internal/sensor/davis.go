package sensor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	DavisAPIV1URL     = "https://api.weatherlink.com/v1/NoaaExt.json"
	DavisAPIV2URL     = "https://api.weatherlink.com/v2"
	DavisDashboardURL = "https://www.weatherlink.com/embeddablePage/summaryData"
	HTTPUserAgent     = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

type DavisSensor interface {
	FetchLatest() ([]DavisCurrentObservation, error)
}

type DavisFactory func(cred DavisAPICredentials, sleepDuration time.Duration) DavisSensor

type DavisAPICredentials struct {
	User     string
	Pass     string
	APIToken string

	APIKey    string
	APISecret string

	StnUUID string
}

type Davis struct {
	api    DavisAPICredentials
	client Fetcher
	sleep  time.Duration
}

type DavisCurrentObservation struct {
	Rr            pgtype.Float4      `json:"rain"`
	Temp          pgtype.Float4      `json:"temp"`
	Rh            pgtype.Float4      `json:"rh"`
	Wdir          pgtype.Float4      `json:"wdir"`
	Wspd          pgtype.Float4      `json:"wspd"`
	Wspdx         pgtype.Float4      `json:"gust"`
	Srad          pgtype.Float4      `json:"srad"`
	Pres          pgtype.Float4      `json:"mslp"`
	Tn            pgtype.Float4      `json:"tn"`
	Tx            pgtype.Float4      `json:"tx"`
	Hi            pgtype.Float4      `json:"hi"`
	RainAccum     pgtype.Float4      `json:"rain_accum"`
	TnTimestamp   pgtype.Timestamptz `json:"tn_timestamp"`
	TxTimestamp   pgtype.Timestamptz `json:"tx_timestamp"`
	GustTimestamp pgtype.Timestamptz `json:"gust_timestamp"`
	Timestamp     pgtype.Timestamptz `json:"timestamp"`
}

type davisRawCurrentResponse interface {
	ToDavisCurrentObservation() *DavisCurrentObservation
}

func NewDavis(apiCredentials DavisAPICredentials, sleep time.Duration) *Davis {
	client := &http.Client{Timeout: 10 * time.Second}
	return &Davis{
		api:    apiCredentials,
		client: client,
		sleep:  sleep,
	}
}

func (d Davis) FetchLatest() ([]DavisCurrentObservation, error) {
	isV1 := d.api.User != "" && d.api.Pass != "" && d.api.APIToken != ""
	isV2 := d.api.APIKey != "" && d.api.APISecret != ""
	isDashboard := d.api.StnUUID != ""

	if !isV1 && !isV2 && !isDashboard {
		return nil, fmt.Errorf("API credentials are invalid")
	}

	var (
		apiURL  string
		qParams url.Values
	)

	if isV2 {
		apiURL = DavisAPIV2URL + "/stations"
		qParams = url.Values{
			"api-key": {d.api.APIKey},
		}
	} else if isV1 {
		apiURL = DavisAPIV1URL
		qParams = url.Values{
			"user":     {d.api.User},
			"pass":     {d.api.Pass},
			"apiToken": {d.api.APIToken},
		}
	} else {
		apiURL = fmt.Sprintf("%s/%s", DavisDashboardURL, d.api.StnUUID)
		qParams = url.Values{}
	}

	baseURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	baseURL.RawQuery = qParams.Encode()
	encodedURL := baseURL.String()

	req, err := http.NewRequest("GET", encodedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", HTTPUserAgent)

	if isV2 {
		req.Header.Set("X-Api-Secret", d.api.APISecret)
	}

	time.Sleep(d.sleep)

	res, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if isV2 {
		var rawStations davisRawStationsResponseV2
		err = json.NewDecoder(res.Body).Decode(&rawStations)
		if err != nil {
			return nil, err
		}

		obsSlice := make([]DavisCurrentObservation, 0)
		for _, rawStn := range rawStations.Stations {
			apiURL = fmt.Sprintf("%s/current/%d", DavisAPIV2URL, rawStn.StationID)
			qParams = url.Values{
				"api-key": {d.api.APIKey},
			}

			baseURL, err := url.Parse(apiURL)
			if err != nil {
				return nil, err
			}
			baseURL.RawQuery = qParams.Encode()
			encodedURL := baseURL.String()

			req, err := http.NewRequest("GET", encodedURL, nil)
			if err != nil {
				return nil, err
			}

			req.Header.Set("User-Agent", HTTPUserAgent)
			req.Header.Set("X-Api-Secret", d.api.APISecret)

			time.Sleep(d.sleep)

			res, err := d.client.Do(req)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()

			var rawObs davisRawCurrentResponseV2
			err = json.NewDecoder(res.Body).Decode(&rawObs)
			if err != nil {
				return nil, err
			}

			obs := rawObs.ToDavisCurrentObservation()
			if obs != nil {
				obsSlice = append(obsSlice, *obs)
			}
		}
		return obsSlice, nil
	} else if isV1 {
		var rawObs davisRawCurrentResponseV1
		err = json.NewDecoder(res.Body).Decode(&rawObs)
		if err != nil {
			return nil, err
		}

		obs := rawObs.ToDavisCurrentObservation()
		return []DavisCurrentObservation{*obs}, nil
	}
	var rawObs davisRawWeatherDataResponseDashboard
	err = json.NewDecoder(res.Body).Decode(&rawObs)
	if err != nil {
		return nil, err
	}

	obs := rawObs.ToDavisCurrentObservation()
	return []DavisCurrentObservation{*obs}, nil
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
