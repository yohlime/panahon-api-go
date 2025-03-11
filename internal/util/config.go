package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config store all configuration of the application.
// Values are read by viper from a config file or environment variables.
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	GinMode              string        `mapstructure:"GIN_MODE"`
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	MigrationPath        string        `mapstructure:"MIGRATION_PATH"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	CookieDomain         string        `mapstructure:"COOKIE_DOMAIN"`
	CookiePath           string        `mapstructure:"COOKIE_PATH"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	APIBasePath          string        `mapstructure:"API_BASE_PATH"`
	SwagAPIBasePath      string        `mapstructure:"SWAG_API_BASE_PATH"`
	GlabsAppID           string        `mapstructure:"GLABS_APP_ID"`
	GlabsAppSecret       string        `mapstructure:"GLABS_APP_SECRET"`
	EnableConsoleLogging bool          `mapstructure:"ENABLE_CONSOLE_LOGGING"`
	EnableFileLogging    bool          `mapstructure:"ENABLE_FILE_LOGGING"`
	LogLevel             string        `mapstructure:"LOG_LEVEL"`
	LogDirectory         string        `mapstructure:"LOG_DIRECTORY"`
	LogFilename          string        `mapstructure:"LOG_FILENAME"`
	LogMaxSize           int           `mapstructure:"LOG_MAX_SIZE"`
	LogMaxBackups        int           `mapstructure:"LOG_MAX_BACKUPS"`
	LogMaxAge            int           `mapstructure:"LOG_MAX_AGE"`
	CronJobs             []CronJob     `mapstructure:"-"`
	DockerTestPGRepo     string        `mapstructure:"DOCKERTEST_PG_REPO"`
	DockerTestPGTag      string        `mapstructure:"DOCKERTEST_PG_TAG"`
}

type CronJob struct {
	Name     string `mapstructure:"name"`
	Schedule string `mapstructure:"schedule"`
}

// LoadConfig read configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.SetDefault("GinMode", "debug")
	viper.SetDefault("APIBasePath", "/")
	viper.SetDefault("SwagAPIBasePath", "/")
	viper.SetDefault("CookieDomain", "localhost")
	viper.SetDefault("CookiePath", "/")
	viper.SetDefault("LogDirectory", "./logs")
	viper.SetDefault("LogFilename", "log")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	if err = viper.Unmarshal(&config); err != nil {
		return
	}

	viper.SetConfigName("app_conf")
	viper.SetConfigType("yaml")

	if err = viper.ReadInConfig(); err != nil {
		return
	}

	if err = viper.UnmarshalKey("cron_jobs", &config.CronJobs); err != nil {
		return
	}

	return
}
