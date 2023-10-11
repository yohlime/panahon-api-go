package sensor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLufft(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	lufft := RandomLufft()

	testCases := []struct {
		name        string
		buildArg    func() (string, *Lufft, error)
		checkResult func(valStr string, lufft2 *Lufft, err error)
	}{
		{
			name: "Lufft23",
			buildArg: func() (string, *Lufft, error) {
				valStr := lufft.String(23)
				lufft2, err := NewLufftFromString(valStr)

				return valStr, lufft2, err
			},
			checkResult: func(valStr string, lufft2 *Lufft, err error) {
				require.NoError(t, err)
				requireLufftEqual(t, lufft, *lufft2)

				require.Equal(t, lufft2.Health.Message.String, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference.Int32, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg.String, "")
				require.Equal(t, lufft2.Health.DataCount.Int32, int32(10))
				require.Equal(t, lufft2.Health.DataStatus.String, "1111111111")
			},
		},
		{
			name: "Lufft19",
			buildArg: func() (string, *Lufft, error) {
				valStr := lufft.String(19)
				lufft2, err := NewLufftFromString(valStr)

				return valStr, lufft2, err
			},
			checkResult: func(valStr string, lufft2 *Lufft, err error) {
				require.NoError(t, err)
				requireLufftEqual(t, lufft, *lufft2)

				require.False(t, lufft2.Health.Vb2.Valid)
				require.False(t, lufft2.Health.Curr.Valid)
				require.False(t, lufft2.Health.Bp2.Valid)
				require.False(t, lufft2.Health.Cm.Valid)

				require.Equal(t, lufft2.Health.Message.String, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference.Int32, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg.String, "")
				require.Equal(t, lufft2.Health.DataCount.Int32, int32(10))
				require.Equal(t, lufft2.Health.DataStatus.String, "1111111111")
			},
		},
		{
			name: "Lufft24",
			buildArg: func() (string, *Lufft, error) {
				valStr := lufft.String(24)
				lufft2, err := NewLufftFromString(valStr)

				return valStr, lufft2, err
			},
			checkResult: func(valStr string, lufft2 *Lufft, err error) {
				require.NoError(t, err)
				requireLufftEqual(t, lufft, *lufft2)

				require.Equal(t, lufft2.Health.Message.String, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference.Int32, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg.String, "")
				require.Equal(t, lufft2.Health.DataCount.Int32, int32(10))
				require.Equal(t, lufft2.Health.DataStatus.String, "1111111111")
			},
		},
		{
			name: "Lufft20",
			buildArg: func() (string, *Lufft, error) {
				valStr := lufft.String(20)
				lufft2, err := NewLufftFromString(valStr)

				return valStr, lufft2, err
			},
			checkResult: func(valStr string, lufft2 *Lufft, err error) {
				require.NoError(t, err)
				requireLufftEqual(t, lufft, *lufft2)

				require.False(t, lufft2.Health.Vb2.Valid)
				require.False(t, lufft2.Health.Curr.Valid)
				require.False(t, lufft2.Health.Bp2.Valid)
				require.False(t, lufft2.Health.Cm.Valid)

				require.Equal(t, lufft2.Health.Message.String, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference.Int32, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg.String, "")
				require.Equal(t, lufft2.Health.DataCount.Int32, int32(10))
				require.Equal(t, lufft2.Health.DataStatus.String, "1111111111")
			},
		},
		{
			name: "InvalidString",
			buildArg: func() (string, *Lufft, error) {
				valStr := "invalid string"
				lufft2, err := NewLufftFromString(valStr)

				return valStr, lufft2, err
			},
			checkResult: func(valStr string, lufft2 *Lufft, err error) {
				require.Error(t, err)
				require.Empty(t, lufft2)
			},
		},
		{
			name: "InvalidLength",
			buildArg: func() (string, *Lufft, error) {
				valStr := lufft.String(23)
				valStr = strings.Join((strings.Split(valStr, "+"))[:15], "+")
				lufft2, err := NewLufftFromString(valStr)
				return valStr, lufft2, err
			},
			checkResult: func(valStr string, lufft2 *Lufft, err error) {
				require.Error(t, err)
				require.Empty(t, lufft2)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			valStr, lufft2, err := tc.buildArg()
			tc.checkResult(valStr, lufft2, err)
		})
	}
}

func requireLufftEqual(t *testing.T, l, l2 Lufft) {
	require.InDelta(t, l.Obs.Temp.Float32, l2.Obs.Temp.Float32, 0.01)
	require.InDelta(t, l.Obs.Rh.Float32, l2.Obs.Rh.Float32, 0.01)
	require.InDelta(t, l.Obs.Pres.Float32, l2.Obs.Pres.Float32, 0.01)
	require.InDelta(t, l.Obs.Wspd.Float32, l2.Obs.Wspd.Float32, 0.01)
	require.InDelta(t, l.Obs.Wspdx.Float32, l2.Obs.Wspdx.Float32, 0.01)
	require.InDelta(t, l.Obs.Wdir.Float32, l2.Obs.Wdir.Float32, 0.01)
	require.InDelta(t, l.Obs.Srad.Float32, l2.Obs.Srad.Float32, 0.01)
	require.InDelta(t, l.Obs.Td.Float32, l2.Obs.Td.Float32, 0.01)
	require.InDelta(t, l.Obs.Wchill.Float32, l2.Obs.Wchill.Float32, 0.01)
	require.InDelta(t, l.Obs.Rr.Float32, l2.Obs.Rr.Float32, 1)
	require.InDelta(t, l.Health.Vb1.Float32, l2.Health.Vb1.Float32, 0.01)
	require.InDelta(t, l.Health.Bp1.Float32, l2.Health.Bp1.Float32, 0.01)
	require.Equal(t, l.Health.Ss, l2.Health.Ss)
	require.InDelta(t, l.Health.TempArq.Float32, l2.Health.TempArq.Float32, 0.01)
	require.InDelta(t, l.Health.RhArq.Float32, l2.Health.RhArq.Float32, 0.01)
	require.Equal(t, l.Health.Fpm, l2.Health.Fpm)

	if l2.Health.Vb2.Valid {
		require.InDelta(t, l.Health.Vb2.Float32, l2.Health.Vb2.Float32, 0.01)
	}
	if l2.Health.Curr.Valid {
		require.InDelta(t, l.Health.Curr.Float32, l2.Health.Curr.Float32, 0.01)
	}
	if l2.Health.Bp2.Valid {
		require.InDelta(t, l.Health.Bp2.Float32, l2.Health.Bp2.Float32, 0.01)
	}
	if l2.Health.Cm.Valid {
		require.Equal(t, l.Health.Cm, l2.Health.Cm)
	}
}
