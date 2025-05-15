package sensor

import (
	"fmt"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/stretchr/testify/require"
)

func TestMisol(t *testing.T) {
	timeNow := time.Now().Truncate(time.Second)
	testCases := []struct {
		name        string
		buildArg    func() (string, *Misol, error)
		checkResult func(valStr string, m2 *Misol, err error)
	}{
		{
			name: "Default",
			buildArg: func() (string, *Misol, error) {
				valStr := fmt.Sprintf("75112112108101,123.8854,10.3157,%d,31,91,1007,6.7,13.4,105,1718,77,30.0001,2,23,3.7,3.8,0.012,17.8,0.065,50,10,25.2,54.4,0,90,1", timeNow.Unix())
				misol2, err := NewMisolFromString(valStr)
				return valStr, misol2, err
			},
			checkResult: func(valStr string, m2 *Misol, err error) {
				require.NoError(t, err)

				m1 := Misol{
					Obs: StationObservation{
						Temp:               util.ToRef(float32(31.0)),
						Rh:                 util.ToRef(float32(91)),
						Pres:               util.ToRef(float32(1007)),
						Wspd:               util.ToRef(float32(6.7 / 3.6)),
						Wspdx:              util.ToRef(float32(13.4 / 3.6)),
						Wdir:               util.ToRef(float32(105)),
						Srad:               util.ToRef(float32(1718)),
						Td:                 util.ToRef(float32(77)),
						Wchill:             util.ToRef(float32(30.001)),
						RainTips:           util.ToRef(int32(2)),
						RainCumulativeTips: util.ToRef(int32(23)),
						Timestamp:          timeNow,
					},
					Health: StationHealth{
						Vb1:       util.ToRef(float32(3.7)),
						Vb2:       util.ToRef(float32(3.8)),
						Curr:      util.ToRef(float32(0.012)),
						Bp1:       util.ToRef(float32(17.8)),
						Bp2:       util.ToRef(float32(0.065)),
						TempArq:   util.ToRef(float32(25.2)),
						RhArq:     util.ToRef(float32(54.4)),
						Timestamp: timeNow,
					},
				}

				requireMisolEqual(t, m1, *m2)

				require.Equal(t, m2.Health.Message, valStr)
				requireMisolHealthNoError(t, *m2)
			},
		},
		{
			name: "InvalidString",
			buildArg: func() (string, *Misol, error) {
				valStr := "invalid string"
				misol2, err := NewMisolFromString(valStr)

				return valStr, misol2, err
			},
			checkResult: func(valStr string, m2 *Misol, err error) {
				require.Error(t, err)
				require.Empty(t, m2)
			},
		},
		{
			name: "InvalidLength",
			buildArg: func() (string, *Misol, error) {
				valStr := fmt.Sprintf("75112112108101,123.8854,10.3157,%d,31,91,1007,6.7,13.4,105,1718,77,23,3.7,3.8,0.012,17.8,50,10,25.2,54.4,0,90,1", timeNow.Unix())
				misol2, err := NewMisolFromString(valStr)

				return valStr, misol2, err
			},
			checkResult: func(valStr string, m2 *Misol, err error) {
				require.Error(t, err)
				require.Empty(t, m2)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			valStr, misol2, err := tc.buildArg()
			tc.checkResult(valStr, misol2, err)
		})
	}
}

func requireMisolEqual(t *testing.T, m1, m2 Misol) {
	require.InDelta(t, *m1.Obs.Temp, *m2.Obs.Temp, 0.01, "Temp value mismatch")
	require.InDelta(t, *m1.Obs.Rh, *m2.Obs.Rh, 0.01, "Rh value mismatch")
	require.InDelta(t, *m1.Obs.Pres, *m2.Obs.Pres, 0.01, "Pres value mismatch")
	require.InDelta(t, *m1.Obs.Wspd, *m2.Obs.Wspd, 0.01, "Wspd value mismatch")
	require.InDelta(t, *m1.Obs.Wspdx, *m2.Obs.Wspdx, 0.01, "Wspdx value mismatch")
	require.InDelta(t, *m1.Obs.Wdir, *m2.Obs.Wdir, 0.01, "Wdir value mismatch")
	require.InDelta(t, *m1.Obs.Srad, *m2.Obs.Srad, 0.01, "Srad value mismatch")
	require.InDelta(t, *m1.Obs.Td, *m2.Obs.Td, 0.01, "Td value mismatch")
	require.InDelta(t, *m1.Obs.Wchill, *m2.Obs.Wchill, 0.01, "Wchill value mismatch")
	require.InDelta(t, *m1.Obs.RainTips, *m2.Obs.RainTips, 1, "Rain tips value mismatch")
	require.InDelta(t, *m1.Obs.RainCumulativeTips, *m2.Obs.RainCumulativeTips, 1, "Rain cumulative tips value mismatch")
	require.Equal(t, m1.Obs.Timestamp, m2.Obs.Timestamp, "Timestamp mismatch")

	require.InDelta(t, *m1.Health.Vb1, *m2.Health.Vb1, 0.01, "Vb1 value mismatch")
	require.InDelta(t, *m1.Health.Vb2, *m2.Health.Vb2, 0.01, "Vb2 value mismatch")
	require.InDelta(t, *m1.Health.Curr, *m2.Health.Curr, 0.01, "Curr value mismatch")
	require.InDelta(t, *m1.Health.Bp1, *m2.Health.Bp1, 0.01, "Bp1 value mismatch")
	require.InDelta(t, *m1.Health.Bp2, *m2.Health.Bp2, 0.01, "Bp2 value mismatch")
	// require.Equal(t, *m1.Health.Ss, *m2.Health.Ss, "Ss value mismatch")
	require.InDelta(t, *m1.Health.TempArq, *m2.Health.TempArq, 0.01, "TempArq value mismatch")
	require.InDelta(t, *m1.Health.RhArq, *m2.Health.RhArq, 0.01, "RhArq value mismatch")
	// require.Equal(t, m1.Health.Fpm, m2.Health.Fpm, "Fpm value mismatch")
	//
	// if m2.Health.Cm != "" {
	// 	require.Equal(t, m1.Health.Cm, m2.Health.Cm, "Cm value mismatch")
	// }
}

func requireMisolHealthNoError(t *testing.T, m Misol) {
	require.Equal(t, m.Health.MinutesDifference, int32(0))
	require.Equal(t, m.Health.ErrorMsg, "")
	require.Equal(t, m.Health.DataCount, int32(11))
	require.Equal(t, m.Health.DataStatus, "11111111111")
}
