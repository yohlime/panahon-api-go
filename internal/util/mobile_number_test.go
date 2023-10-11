package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMobileNumber(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		checkResult func(inStr, outStr string, ok bool)
	}{
		{
			name:  "##########",
			input: "1234567890",
			checkResult: func(inStr, outStr string, ok bool) {
				require.True(t, ok)
				require.Equal(t, inStr, outStr[2:])
			},
		},
		{
			name:  "0##########",
			input: "01234567890",
			checkResult: func(inStr, outStr string, ok bool) {
				require.True(t, ok)
				require.Equal(t, inStr[1:], outStr[2:])
			},
		},
		{
			name:  "63##########",
			input: "631234567890",
			checkResult: func(inStr, outStr string, ok bool) {
				require.True(t, ok)
				require.Equal(t, inStr, outStr)
			},
		},
		{
			name:  "+63##########",
			input: "+631234567890",
			checkResult: func(inStr, outStr string, ok bool) {
				require.True(t, ok)
				require.Equal(t, inStr[1:], outStr)
			},
		},
		{
			name:  "NoMatch",
			input: "as1234qwer",
			checkResult: func(inStr, outStr string, ok bool) {
				require.False(t, ok)
				require.Len(t, outStr, 0)
			},
		},
		{
			name:  "WrongLength",
			input: "1234567",
			checkResult: func(inStr, outStr string, ok bool) {
				require.False(t, ok)
				require.Len(t, outStr, 0)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			outStr, ok := ParseMobileNumber(tc.input)
			tc.checkResult(tc.input, outStr, ok)
		})
	}
}
