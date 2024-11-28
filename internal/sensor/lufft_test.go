package sensor

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLufft(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	lufft := RandomLufft(time.Now())

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
				requireLufftHealthNoError(t, *lufft2)
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

				require.Nil(t, lufft2.Health.Vb2)
				require.Nil(t, lufft2.Health.Curr)
				require.Nil(t, lufft2.Health.Bp2)
				require.Empty(t, lufft2.Health.Cm)

				requireLufftEqual(t, lufft, *lufft2)

				require.Equal(t, lufft2.Health.Message, valStr)
				requireLufftHealthNoError(t, *lufft2)
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
				requireLufftHealthNoError(t, *lufft2)
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

				require.Nil(t, lufft2.Health.Vb2)
				require.Nil(t, lufft2.Health.Curr)
				require.Nil(t, lufft2.Health.Bp2)
				require.Empty(t, lufft2.Health.Cm)

				requireLufftEqual(t, lufft, *lufft2)

				require.Equal(t, lufft2.Health.Message, valStr)
				requireLufftHealthNoError(t, *lufft2)
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
	require.InDelta(t, *l.Obs.Temp, *l2.Obs.Temp, 0.01, "Temp value mismatch")
	require.InDelta(t, *l.Obs.Rh, *l2.Obs.Rh, 0.01, "Rh value mismatch")
	require.InDelta(t, *l.Obs.Pres, *l2.Obs.Pres, 0.01, "Pres value mismatch")
	require.InDelta(t, *l.Obs.Wspd, *l2.Obs.Wspd, 0.01, "Wspd value mismatch")
	require.InDelta(t, *l.Obs.Wspdx, *l2.Obs.Wspdx, 0.01, "Wspdx value mismatch")
	require.InDelta(t, *l.Obs.Wdir, *l2.Obs.Wdir, 0.01, "Wdir value mismatch")
	require.InDelta(t, *l.Obs.Srad, *l2.Obs.Srad, 0.01, "Srad value mismatch")
	require.InDelta(t, *l.Obs.Td, *l2.Obs.Td, 0.01, "Td value mismatch")
	require.InDelta(t, *l.Obs.Wchill, *l2.Obs.Wchill, 0.01, "Wchill value mismatch")
	require.InDelta(t, *l.Obs.Rr, *l2.Obs.Rr, 1, "Rr value mismatch")
	require.InDelta(t, *l.Health.Vb1, *l2.Health.Vb1, 0.01, "Vb1 value mismatch")
	require.InDelta(t, *l.Health.Bp1, *l2.Health.Bp1, 0.01, "Bp1 value mismatch")
	require.Equal(t, *l.Health.Ss, *l2.Health.Ss, "Ss value mismatch")
	require.InDelta(t, *l.Health.TempArq, *l2.Health.TempArq, 0.01, "TempArq value mismatch")
	require.InDelta(t, *l.Health.RhArq, *l2.Health.RhArq, 0.01, "RhArq value mismatch")
	require.Equal(t, l.Health.Fpm, l2.Health.Fpm, "Fpm value mismatch")

	if l2.Health.Vb2 != nil {
		require.InDelta(t, *l.Health.Vb2, *l2.Health.Vb2, 0.01, "Vb2 value mismatch")
	}
	if l2.Health.Curr != nil {
		require.InDelta(t, *l.Health.Curr, *l2.Health.Curr, 0.01, "Curr value mismatch")
	}
	if l2.Health.Bp2 != nil {
		require.InDelta(t, *l.Health.Bp2, *l2.Health.Bp2, 0.01, "Bp2 value mismatch")
	}
	if l2.Health.Cm != "" {
		require.Equal(t, l.Health.Cm, l2.Health.Cm, "Cm value mismatch")
	}
}

func requireLufftHealthNoError(t *testing.T, l Lufft) {
	require.Equal(t, l.Health.MinutesDifference, int32(0))
	require.Equal(t, l.Health.ErrorMsg, "")
	require.Equal(t, l.Health.DataCount, int32(10))
	require.Equal(t, l.Health.DataStatus, "1111111111")
}
