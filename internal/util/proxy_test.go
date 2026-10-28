package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTrustedProxies(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		proxies, err := ParseTrustedProxies("  ")
		require.NoError(t, err)
		require.Nil(t, proxies)
	})

	t.Run("IPsAndCIDRs", func(t *testing.T) {
		proxies, err := ParseTrustedProxies("127.0.0.1, ::1, 10.0.0.0/8, 192.168.1.0/24")
		require.NoError(t, err)
		require.Equal(t, []string{"127.0.0.1", "::1", "10.0.0.0/8", "192.168.1.0/24"}, proxies)
	})

	t.Run("Invalid", func(t *testing.T) {
		_, err := ParseTrustedProxies("proxy.internal")
		require.Error(t, err)
	})
}
