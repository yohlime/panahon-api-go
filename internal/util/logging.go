package util

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logLevelMap = map[string]zerolog.Level{
	"trace": zerolog.TraceLevel,
	"debug": zerolog.DebugLevel,
	"info":  zerolog.InfoLevel,
	"warn":  zerolog.WarnLevel,
	"error": zerolog.ErrorLevel,
	"fatal": zerolog.FatalLevel,
	"panic": zerolog.PanicLevel,
}

func NewLogger(config Config) *zerolog.Logger {
	var writers []io.Writer

	if config.EnableConsoleLogging {
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stderr})
	}
	if config.EnableFileLogging {
		writers = append(writers, newRollingFile(config))
	}
	mw := io.MultiWriter(writers...)

	zerolog.SetGlobalLevel(validateLogLevel(config.LogLevel))
	logger := zerolog.New(mw).With().Timestamp().Logger()

	logger.Info().
		Bool("fileLogging", config.EnableFileLogging).
		Str("logDirectory", config.LogDirectory).
		Str("logFileName", config.LogFilename).
		Int("logMaxSizeMB", config.LogMaxSize).
		Int("logMaxBackups", config.LogMaxBackups).
		Int("logMaxAgeInDays", config.LogMaxAge).
		Msg("logging configured")

	return &logger
}

func newRollingFile(config Config) io.Writer {
	if err := os.MkdirAll(config.LogDirectory, 0744); err != nil {
		log.Error().Err(err).Str("path", config.LogDirectory).Msg("can't create log directory")
		return nil
	}

	return &lumberjack.Logger{
		Filename:   path.Join(config.LogDirectory, config.LogFilename),
		MaxBackups: config.LogMaxBackups, // files
		MaxSize:    config.LogMaxSize,    // megabytes
		MaxAge:     config.LogMaxAge,     // days
	}
}

func validateLogLevel(level string) zerolog.Level {
	if logLevel, exists := logLevelMap[level]; exists {
		return logLevel
	}
	fmt.Printf("Invalid log level: %s. Defaulting to info level.\n", level)
	return zerolog.InfoLevel
}
