package sensor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	mocksensor "github.com/emiliogozo/panahon-api-go/internal/mocks"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDavis(t *testing.T) {
	testCases := []struct {
		name           string
		apiCredentials DavisAPICredentials
		checkInstance  func(sensor *Davis)
	}{
		{
			name: "Default",
			apiCredentials: DavisAPICredentials{
				User:     "testdavisUser",
				Pass:     "testdav!sPAss",
				APIToken: "123DAV15890456ZXCARSLUY",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Equal(t, "testdavisUser", sensor.api.User)
				require.Equal(t, "testdav!sPAss", sensor.api.Pass)
				require.Equal(t, "123DAV15890456ZXCARSLUY", sensor.api.APIToken)
			},
		},
		{
			name: "V2",
			apiCredentials: DavisAPICredentials{
				APIKey:    "123DAV15890456ZXCARSLUY",
				APISecret: "xxxx123DAV15890456ZXCARSLUYxxxx",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Equal(t, "123DAV15890456ZXCARSLUY", sensor.api.APIKey)
				require.Equal(t, "xxxx123DAV15890456ZXCARSLUYxxxx", sensor.api.APISecret)
			},
		},
		{
			name: "Dashboard",
			apiCredentials: DavisAPICredentials{
				StnUUID: "a72efc82c04d43b6801d7d4de46aa79a",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Equal(t, "a72efc82c04d43b6801d7d4de46aa79a", sensor.api.StnUUID)
			},
		},
		{
			name: "NoUser",
			apiCredentials: DavisAPICredentials{
				Pass:     "testdav!sPAss",
				APIToken: "123DAV15890456ZXCARSLUY",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Empty(t, sensor.api.User)
				require.Equal(t, "testdav!sPAss", sensor.api.Pass)
				require.Equal(t, "123DAV15890456ZXCARSLUY", sensor.api.APIToken)
			},
		},
		{
			name: "NoPass",
			apiCredentials: DavisAPICredentials{
				User:     "testdavisUser",
				APIToken: "123DAV15890456ZXCARSLUY",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Equal(t, "testdavisUser", sensor.api.User)
				require.Empty(t, sensor.api.Pass)
				require.Equal(t, "123DAV15890456ZXCARSLUY", sensor.api.APIToken)
			},
		},
		{
			name: "NoAPIToken",
			apiCredentials: DavisAPICredentials{
				User: "testdavisUser",
				Pass: "testdav!sPAss",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Equal(t, "testdavisUser", sensor.api.User)
				require.Equal(t, "testdav!sPAss", sensor.api.Pass)
				require.Empty(t, sensor.api.APIToken)
			},
		},
		{
			name: "NoAPIKey",
			apiCredentials: DavisAPICredentials{
				APISecret: "xxxx123DAV15890456ZXCARSLUYxxxx",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Empty(t, sensor.api.APIKey)
				require.Equal(t, "xxxx123DAV15890456ZXCARSLUYxxxx", sensor.api.APISecret)
			},
		},
		{
			name: "NoAPISecret",
			apiCredentials: DavisAPICredentials{
				APIKey: "123DAV15890456ZXCARSLUY",
			},
			checkInstance: func(sensor *Davis) {
				require.NotNil(t, sensor)
				require.Equal(t, "123DAV15890456ZXCARSLUY", sensor.api.APIKey)
				require.Empty(t, sensor.api.APISecret)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			testSensor := NewDavis(tc.apiCredentials, 10)
			tc.checkInstance(testSensor)
		})
	}
}

func TestFetchLatest(t *testing.T) {
	testCases := []struct {
		name          string
		api           DavisAPICredentials
		builStubs     func(client *mocksensor.MockFetcher) []davisRawCurrentResponse
		checkResponse func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error)
	}{
		{
			name: "Default",
			api: DavisAPICredentials{
				User:     "testuser001",
				Pass:     "secrEtp@s$",
				APIToken: "qwfparst1234ar655",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				rawObs := randomDavisRawResponseV1()
				body, _ := json.Marshal(rawObs)
				bodyReader := bytes.NewReader(body)
				client.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool { return req.Header.Get("User-Agent") != "" })).Return(&http.Response{
					Body: io.NopCloser(bodyReader),
				}, nil)
				return []davisRawCurrentResponse{rawObs}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.NoError(t, err)
				requireDavisEqual(t, rawObsSlice, obsSlice)
			},
		},
		{
			name: "V2",
			api: DavisAPICredentials{
				APIKey:    "123DAV15890456ZXCARSLUY",
				APISecret: "xxxx123DAV15890456ZXCARSLUYxxxx",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				rawStns := davisRawStationsResponseV2{
					Stations: []davisRawStationResponseV2{
						{
							StationID: 111,
						},
						{
							StationID: 121,
						},
						{
							StationID: 511,
						},
					},
				}
				body, _ := json.Marshal(rawStns)
				bodyReader := bytes.NewReader(body)
				client.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool { return req.Header.Get("User-Agent") != "" })).Return(&http.Response{
					Body: io.NopCloser(bodyReader),
				}, nil).Once()
				rawObsSlice := make([]davisRawCurrentResponse, 0)
				for range rawStns.Stations {
					rawObs := randomDavisRawResponseV2()
					rawObsSlice = append(rawObsSlice, rawObs)
					body, _ := json.Marshal(rawObs)
					bodyReader := bytes.NewReader(body)
					client.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool { return req.Header.Get("User-Agent") != "" })).Return(&http.Response{
						Body: io.NopCloser(bodyReader),
					}, nil).Once()
				}
				return rawObsSlice
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.NoError(t, err)
				requireDavisEqual(t, rawObsSlice, obsSlice)
			},
		},
		{
			name: "Dashboard",
			api: DavisAPICredentials{
				StnUUID: "a72efc82c04d43b6801d7d4de46aa79a",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				rawObs := randomDavisRawWeatherDataResponseDashboard()
				body, _ := json.Marshal(rawObs)
				bodyReader := bytes.NewReader(body)
				client.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool { return req.Header.Get("User-Agent") != "" })).Return(&http.Response{
					Body: io.NopCloser(bodyReader),
				}, nil)
				return []davisRawCurrentResponse{rawObs}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.NoError(t, err)
				requireDavisEqual(t, rawObsSlice, obsSlice)
			},
		},
		{
			name: "MissingUserParam",
			api: DavisAPICredentials{
				Pass:     "secrEtp@s$",
				APIToken: "qwfparst1234ar655",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				return []davisRawCurrentResponse{}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obsSlice)
			},
		},
		{
			name: "MissingPassParam",
			api: DavisAPICredentials{
				User:     "testuser001",
				APIToken: "qwfparst1234ar655",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				return []davisRawCurrentResponse{}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obsSlice)
			},
		},
		{
			name: "MissingApiToken",
			api: DavisAPICredentials{
				User: "testuser001",
				Pass: "secrEtp@s$",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				return []davisRawCurrentResponse{}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obsSlice)
			},
		},
		{
			name: "MissingApiSecret",
			api: DavisAPICredentials{
				APIKey: "123DAV15890456ZXCARSLUY",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				return []davisRawCurrentResponse{}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obsSlice)
			},
		},
		{
			name: "MissingApiKey",
			api: DavisAPICredentials{
				APISecret: "xxxx123DAV15890456ZXCARSLUYxxxx",
			},
			builStubs: func(client *mocksensor.MockFetcher) []davisRawCurrentResponse {
				return []davisRawCurrentResponse{}
			},
			checkResponse: func(client *mocksensor.MockFetcher, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obsSlice)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			testFetcher := mocksensor.NewMockFetcher(t)
			testSensor := &Davis{
				api:    tc.api,
				client: testFetcher,
				sleep:  0,
			}

			rawObsSlice := tc.builStubs(testFetcher)
			obs, err := testSensor.FetchLatest()

			tc.checkResponse(testFetcher, rawObsSlice, obs, err)
		})
	}
}

func requireDavisEqual(t *testing.T, rawObsSlice []davisRawCurrentResponse, obsSlice []DavisCurrentObservation) {
	for i, rawObs := range rawObsSlice {
		switch v := rawObs.(type) {
		case davisRawCurrentResponseV1:
			requireDavisEqualV1(t, v, obsSlice[i])
		case davisRawCurrentResponseV2:
			requireDavisEqualV2(t, v, obsSlice[i])
		case davisRawWeatherDataResponseDashboard:
			requireDavisEqualDashboard(t, v, obsSlice[i])
		}
	}
}

func requireDavisEqualV1(t *testing.T, rawObs davisRawCurrentResponseV1, obs DavisCurrentObservation) {
	if rawObs.PressureMb != nil {
		require.InDelta(t, *rawObs.PressureMb, obs.Pres.Float32, 0.001)
	}
	if rawObs.Rh != nil {
		require.InDelta(t, *rawObs.Rh, obs.Rh.Float32, 0.001)
	}
	if rawObs.TempC != nil {
		require.InDelta(t, *rawObs.TempC, obs.Temp.Float32, 0.001)
	}
	if rawObs.WindDeg != nil {
		require.InDelta(t, *rawObs.WindDeg, obs.Wdir.Float32, 0.001)
	}
	if rawObs.WindMPH != nil {
		require.InDelta(t, *rawObs.WindMPH*0.44704, obs.Wspd.Float32, 0.001)
	}
	if rawObs.Obs.WindDayHighMPH != nil {
		require.InDelta(t, *rawObs.Obs.WindDayHighMPH*0.44704, obs.Wspdx.Float32, 0.001)
	}
	require.Equal(t, rawObs.Obs.TempDayHighTime, datetimeToTimeStr(obs.TxTimestamp.Time))
	require.Equal(t, rawObs.Obs.TempDayLowTime, datetimeToTimeStr(obs.TnTimestamp.Time))
	require.Equal(t, rawObs.Obs.WindDayHighTime, datetimeToTimeStr(obs.GustTimestamp.Time))
	require.Equal(t, rawObs.Time, obs.Timestamp.Time.Format("Mon, 02 Jan 2006 15:04:05 -0700"))
}

func requireDavisEqualV2(t *testing.T, rawObs davisRawCurrentResponseV2, obs DavisCurrentObservation) {
	rawObsData := rawObs.Sensors[0].Data[0]
	if rawObsData.Bar != nil {
		require.InDelta(t, *rawObsData.Bar, obs.Pres.Float32, 0.001)
	}
	if rawObsData.HumOut != nil {
		require.InDelta(t, *rawObsData.HumOut, obs.Rh.Float32, 0.001)
	}
	if rawObsData.TempOut != nil {
		require.InDelta(t, util.FahrenheitToCelsius(*rawObsData.TempOut), obs.Temp.Float32, 0.001)
	}
}

func requireDavisEqualDashboard(t *testing.T, rawObs davisRawWeatherDataResponseDashboard, obs DavisCurrentObservation) {
	for _, cur := range rawObs.CurrConditionValues {
		if cur.SensorDataTypeID == nil {
			continue
		}

		switch *cur.SensorDataTypeID {
		case 7:
			require.InDelta(t, *cur.Value, obs.Temp.Float32, 0.001)

		case 12:
			require.InDelta(t, *cur.Value, obs.Hi.Float32, 0.001)

		case 22:
			require.InDelta(t, *cur.Value, obs.Rr.Float32, 0.001)

		default:
			continue
		}
	}

	for _, hl := range rawObs.HighLowValues {
		if hl.SensorDataTypeID == nil {
			continue
		}
		switch *hl.SensorDataTypeID {
		case 57:
			require.InDelta(t, *hl.Value, obs.Tx.Float32, 0.001)

		default:
			continue
		}
	}
}

func datetimeToTimeStr(dt time.Time) string {
	return fmt.Sprintf("%s:%02d%s", dt.Format("3"), dt.Minute(), dt.Format("pm"))
}

func randomDavisRawResponseV1() davisRawCurrentResponseV1 {
	return davisRawCurrentResponseV1{
		Location:   util.RandomString(24),
		Lat:        util.RandomFloatPtr[float32](4.0, 22.0),
		Lon:        util.RandomFloatPtr[float32](114.0, 121.0),
		PressureMb: util.RandomFloatPtr[float32](990.0, 1100.),
		Rh:         util.RandomFloatPtr[float32](0.0, 100.0),
		TempC:      util.RandomFloatPtr[float32](25.0, 33.0),
		TdC:        util.RandomFloatPtr[float32](25.0, 33.0),
		WindDeg:    util.RandomFloatPtr[float32](0, 360),
		WindMPH:    util.RandomFloatPtr[float32](0.0, 10.0),
		HeatIndexC: util.RandomFloatPtr[float32](30.0, 50.0),
		Obs: davisRawCurrentObservationV1{
			RRInPerHr:       util.RandomFloatPtr[float32](0.0, 5.0),
			RainDayIn:       util.RandomFloatPtr[float32](0.0, 100.0),
			Srad:            util.RandomFloatPtr[float32](0, 400),
			UVIndex:         util.RandomFloatPtr[float32](0.0, 1.0),
			TempDayHighF:    util.RandomFloatPtr[float32](77.0, 104.0),
			TempDayLowF:     util.RandomFloatPtr[float32](60.0, 104.0),
			WindDayHighMPH:  util.RandomFloatPtr[float32](0.0, 20.0),
			TempDayHighTime: randomTimeString(),
			TempDayLowTime:  randomTimeString(),
			WindDayHighTime: randomTimeString(),
		},
		Time: time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"),
	}
}

func randomDavisRawResponseV2() davisRawCurrentResponseV2 {
	return davisRawCurrentResponseV2{
		StationID: util.RandomInt(100000, 999999),
		Sensors: []davisRawCurrentSensorResponseV2{
			{
				LSID: util.RandomInt(100000, 999999),
				Data: []davisRawCurrentDataResponseV2{
					{
						Bar:     util.RandomFloatPtr[float32](990.0, 1100.),
						TempOut: util.RandomFloatPtr[float32](25.0, 33.0),
						HumOut:  util.RandomFloatPtr[float32](0.0, 100.0),
					},
				},
			},
		},
	}
}

func randomDavisRawWeatherDataResponseDashboard() davisRawWeatherDataResponseDashboard {
	tempID, rhID, rainID := 7, 12, 22
	txID := 57
	highTempTime := float32(randomTimeInt())
	return davisRawWeatherDataResponseDashboard{
		CurrConditionValues: []davisRawSensorDataResponseDashboard{
			{
				SensorDataTypeID: &tempID,
				Value:            util.RandomFloatPtr[float32](24.6, 33.7),
			},
			{
				SensorDataTypeID: &rhID,
				Value:            util.RandomFloatPtr[float32](30.0, 100.0),
			},
			{
				SensorDataTypeID: &rainID,
				Value:            util.RandomFloatPtr[float32](0.0, 12.5),
			},
		},
		HighLowValues: []davisRawSensorDataResponseDashboard{
			{
				SensorDataTypeID: &txID,
				Value:            util.RandomFloatPtr[float32](28.5, 39.9),
			},
			{
				SensorDataName: "High Temp Time",
				Value:          &highTempTime,
			},
		},
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

func randomTimeInt() int32 {
	h := util.RandomInt(0, 23)
	m := util.RandomInt(0, 59)

	return int32(h*100 + m)
}
