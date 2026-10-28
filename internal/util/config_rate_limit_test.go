package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func writeRateLimitConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte("ENVIRONMENT=test\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "app_conf.yaml"),
		[]byte("station_status:\n  online_threshold: \"1h\"\ncron_jobs: []\n"),
		0o644,
	))
	return dir
}

func TestLoadConfigRateLimitDefaults(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	config, err := LoadConfig(writeRateLimitConfig(t))
	require.NoError(t, err)
	require.True(t, config.RateLimit.Enabled)
	require.Equal(t, 10, config.RateLimit.LoginPerMinute)
	require.Equal(t, 10, config.RateLimit.RegistrationPerHour)
	require.Equal(t, 10, config.RateLimit.ResetPerMinute)
	require.Equal(t, 120, config.RateLimit.APIKeyPerMinute)
	require.Empty(t, config.TrustedProxies)
}

func TestLoadConfigRateLimitOverrides(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	t.Setenv("RATE_LIMIT_ENABLED", "false")
	t.Setenv("RATE_LIMIT_LOGIN_PER_MINUTE", "7")
	t.Setenv("RATE_LIMIT_REGISTRATION_PER_HOUR", "4")
	t.Setenv("RATE_LIMIT_RESET_PER_MINUTE", "3")
	t.Setenv("RATE_LIMIT_API_KEY_PER_MINUTE", "99")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1,10.0.0.0/8")

	config, err := LoadConfig(writeRateLimitConfig(t))
	require.NoError(t, err)
	require.False(t, config.RateLimit.Enabled)
	require.Equal(t, 7, config.RateLimit.LoginPerMinute)
	require.Equal(t, 4, config.RateLimit.RegistrationPerHour)
	require.Equal(t, 3, config.RateLimit.ResetPerMinute)
	require.Equal(t, 99, config.RateLimit.APIKeyPerMinute)
	require.Equal(t, "127.0.0.1,10.0.0.0/8", config.TrustedProxies)
}
