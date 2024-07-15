package sensor

import (
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
)

func TestLufft(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	var lufft Lufft
	gofakeit.Struct(&lufft)

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

				require.Equal(t, lufft2.Health.Message, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg, "")
				require.Equal(t, lufft2.Health.DataCount, int32(10))
				require.Equal(t, lufft2.Health.DataStatus, "1111111111")
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

				require.Nil(t, lufft2.Health.Vb2)
				require.Nil(t, lufft2.Health.Curr)
				require.Nil(t, lufft2.Health.Bp2)
				require.Nil(t, lufft2.Health.Cm)

				require.Equal(t, lufft2.Health.Message, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg, "")
				require.Equal(t, lufft2.Health.DataCount, int32(10))
				require.Equal(t, lufft2.Health.DataStatus, "1111111111")
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

				require.Equal(t, lufft2.Health.Message, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg, "")
				require.Equal(t, lufft2.Health.DataCount, int32(10))
				require.Equal(t, lufft2.Health.DataStatus, "1111111111")
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

				require.Nil(t, lufft2.Health.Vb2)
				require.Nil(t, lufft2.Health.Curr)
				require.Nil(t, lufft2.Health.Bp2)
				require.Nil(t, lufft2.Health.Cm)

				require.Equal(t, lufft2.Health.Message, valStr)
				require.Equal(t, lufft2.Health.MinutesDifference, int32(0))
				require.Equal(t, lufft2.Health.ErrorMsg, "")
				require.Equal(t, lufft2.Health.DataCount, int32(10))
				require.Equal(t, lufft2.Health.DataStatus, "1111111111")
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
	require.InDelta(t, l.Obs.Temp, l2.Obs.Temp, 0.01)
	require.InDelta(t, l.Obs.Rh, l2.Obs.Rh, 0.01)
	require.InDelta(t, l.Obs.Pres, l2.Obs.Pres, 0.01)
	require.InDelta(t, l.Obs.Wspd, l2.Obs.Wspd, 0.01)
	require.InDelta(t, l.Obs.Wspdx, l2.Obs.Wspdx, 0.01)
	require.InDelta(t, l.Obs.Wdir, l2.Obs.Wdir, 0.01)
	require.InDelta(t, l.Obs.Srad, l2.Obs.Srad, 0.01)
	require.InDelta(t, l.Obs.Td, l2.Obs.Td, 0.01)
	require.InDelta(t, l.Obs.Wchill, l2.Obs.Wchill, 0.01)
	require.InDelta(t, l.Obs.Rr, l2.Obs.Rr, 1)
	require.InDelta(t, l.Health.Vb1, l2.Health.Vb1, 0.01)
	require.InDelta(t, l.Health.Bp1, l2.Health.Bp1, 0.01)
	require.Equal(t, l.Health.Ss, l2.Health.Ss)
	require.InDelta(t, l.Health.TempArq, l2.Health.TempArq, 0.01)
	require.InDelta(t, l.Health.RhArq, l2.Health.RhArq, 0.01)
	require.Equal(t, l.Health.Fpm, l2.Health.Fpm)

	if l2.Health.Vb2 != nil {
		require.InDelta(t, l.Health.Vb2, l2.Health.Vb2, 0.01)
	}
	if l2.Health.Curr != nil {
		require.InDelta(t, l.Health.Curr, l2.Health.Curr, 0.01)
	}
	if l2.Health.Bp2 != nil {
		require.InDelta(t, l.Health.Bp2, l2.Health.Bp2, 0.01)
	}
	if l2.Health.Cm != "" {
		require.Equal(t, l.Health.Cm, l2.Health.Cm)
	}
}
