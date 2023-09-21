package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDateTime(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		checkResult func(inStr string, outTime time.Time, ok bool)
	}{
		{
			name:  "Default",
			input: "2023-01-31T12:45:00+03:00",
			checkResult: func(inStr string, outTime time.Time, ok bool) {
				require.True(t, ok)
				dt, err := time.Parse(time.RFC3339, "2023-01-31T12:45:00+03:00")
				require.NoError(t, err)
				require.WithinDuration(t, outTime, dt, time.Millisecond*1000)
			},
		},
		{
			name:  "NoTimeZone",
			input: "2023-01-31T12:45:00",
			checkResult: func(inStr string, outTime time.Time, ok bool) {
				require.True(t, ok)
				dt, err := time.Parse(time.RFC3339, "2023-01-31T12:45:00+08:00")
				require.NoError(t, err)
				require.WithinDuration(t, outTime, dt, time.Millisecond*1000)
			},
		},
		{
			name:  "DateOnly",
			input: "2023-01-31",
			checkResult: func(inStr string, outTime time.Time, ok bool) {
				require.True(t, ok)
				dt, err := time.Parse(time.RFC3339, "2023-01-31T00:00:00+08:00")
				require.NoError(t, err)
				require.WithinDuration(t, outTime, dt, time.Millisecond*1000)
			},
		},
		{
			name:  "NotDateTime",
			input: "as1234qwer",
			checkResult: func(inStr string, outTime time.Time, ok bool) {
				require.False(t, ok)
				require.True(t, outTime.IsZero())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			dt, ok := ParseDateTime(tc.input)
			tc.checkResult(tc.input, dt, ok)
		})
	}
}
