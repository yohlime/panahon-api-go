package util

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigStationStatus(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte("ENVIRONMENT=test\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "app_conf.yaml"),
		[]byte("station_status:\n  online_threshold: \"2h\"\ncron_jobs:\n  - name: stationStatus\n    schedule: \"*/15 * * * *\"\n"),
		0o644,
	))

	config, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, 2*time.Hour, config.StationStatus.OnlineThreshold)
	require.Len(t, config.CronJobs, 1)
	require.Equal(t, "stationStatus", config.CronJobs[0].Name)
	require.Equal(t, "*/15 * * * *", config.CronJobs[0].Schedule)
}
