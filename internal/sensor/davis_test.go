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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFetchLatest(t *testing.T) {
	rawObs := RandomDavisRawResponse()
	testCases := []struct {
		name          string
		url           string
		builStubs     func(client *mocksensor.MockFetcher)
		checkResponse func(client *mocksensor.MockFetcher, obs *DavisCurrentObservation, err error)
	}{
		{
			name: "Default",
			url:  "https://api.weatherlink.com/v1/NoaaExt.json?user=00DE01CE1D&pass=p@ssW0rd",
			builStubs: func(client *mocksensor.MockFetcher) {
				body, _ := json.Marshal(rawObs)
				bodyReader := bytes.NewReader(body)
				client.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool { return req.Header.Get("User-Agent") != "" })).Return(&http.Response{
					Body: io.NopCloser(bodyReader),
				}, nil)
			},
			checkResponse: func(client *mocksensor.MockFetcher, obs *DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.NoError(t, err)
				requireDavisEqual(t, rawObs, *obs)
			},
		},
		{
			name: "InvalidURL",
			url:  "api.weatherlink.com/v1/NoaaExt.json?user=00DE01CE1D&pass=p@ssW0rd",
			builStubs: func(client *mocksensor.MockFetcher) {
			},
			checkResponse: func(client *mocksensor.MockFetcher, obs *DavisCurrentObservation, err error) {
				client.AssertNotCalled(t, "Do")
				assert.Error(t, err)
				assert.Empty(t, obs)
			},
		},
		{
			name: "MissingUserParam",
			url:  "https://api.weatherlink.com/v1/NoaaExt.json?pass=p@ssW0rd",
			builStubs: func(client *mocksensor.MockFetcher) {
			},
			checkResponse: func(client *mocksensor.MockFetcher, obs *DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obs)
			},
		},
		{
			name: "MissingPassParam",
			url:  "https://api.weatherlink.com/v1/NoaaExt.json?user=00DE01CE1D",
			builStubs: func(client *mocksensor.MockFetcher) {
			},
			checkResponse: func(client *mocksensor.MockFetcher, obs *DavisCurrentObservation, err error) {
				client.AssertExpectations(t)
				assert.Error(t, err)
				assert.Empty(t, obs)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			testFetcher := mocksensor.NewMockFetcher(t)
			testSensor := &Davis{
				Url:    tc.url,
				client: testFetcher,
				sleep:  0,
			}

			tc.builStubs(testFetcher)
			obs, err := testSensor.FetchLatest()

			tc.checkResponse(testFetcher, obs, err)
		})
	}
}

func requireDavisEqual(t *testing.T, rawObs davisRawResponse, obs DavisCurrentObservation) {
	f, err := rawObs.PressureMb.Float64()
	require.NoError(t, err)
	require.InDelta(t, f, obs.Mslp.Float32, 0.001)
	f, err = rawObs.Rh.Float64()
	require.NoError(t, err)
	require.InDelta(t, f, obs.Rh.Float32, 0.001)
	f, err = rawObs.TempC.Float64()
	require.NoError(t, err)
	require.InDelta(t, f, obs.Temp.Float32, 0.001)
	f, err = rawObs.WindDeg.Float64()
	require.NoError(t, err)
	require.InDelta(t, f, obs.Wdir.Float32, 0.001)
	f, err = rawObs.WindMPH.Float64()
	require.NoError(t, err)
	require.InDelta(t, f*0.44704, obs.Wspd.Float32, 0.001)
	f, err = rawObs.Obs.WindDayHighMPH.Float64()
	require.NoError(t, err)
	require.InDelta(t, f*0.44704, obs.Gust.Float32, 0.001)
	require.Equal(t, rawObs.Obs.TempDayHighTime, datetimeToTimeStr(obs.TxTimestamp.Time))
	require.Equal(t, rawObs.Obs.TempDayLowTime, datetimeToTimeStr(obs.TnTimestamp.Time))
	require.Equal(t, rawObs.Obs.WindDayHighTime, datetimeToTimeStr(obs.GustTimestamp.Time))
	require.Equal(t, rawObs.Time, obs.Timestamp.Time.Format("Mon, 02 Jan 2006 15:04:05 -0700"))
}

func datetimeToTimeStr(dt time.Time) string {
	return fmt.Sprintf("%s:%02d%s", dt.Format("3"), dt.Minute(), dt.Format("pm"))
}
