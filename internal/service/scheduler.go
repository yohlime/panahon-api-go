package service

import (
	"context"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog"
)

func ScheduleJobs(ctx context.Context, store db.Store, conf util.Config, logger *zerolog.Logger) {
	s := gocron.NewScheduler(time.Local)

	cronExps := strings.Split(conf.CronJobs, ":")
	numCronExps := len(cronExps)

	if (numCronExps > 0) && (strings.ToLower(cronExps[0]) != "false") {
		if _, err := s.Cron(cronExps[0]).Tag("InsertCurrentObservations").Do(InsertCurrentObservations, ctx, store, logger); err != nil {
			logger.Fatal().Err(err).Str("service", "InsertCurrentObservations").Msg("error scheduling job")
		}
	}

	if (numCronExps > 1) && (strings.ToLower(cronExps[1]) != "false") {
    davisFactory := func (stnUrl string, sleepDuration time.Duration) sensor.DavisSensor {
			return sensor.NewDavis(stnUrl, sleepDuration)
    }
		if _, err := s.Cron(cronExps[1]).Tag("InsertCurrentDavisObservations").Do(InsertCurrentDavisObservations, ctx, davisFactory, store, logger); err != nil {
			logger.Fatal().Err(err).Str("service", "InsertCurrentDavisObservations").Msg("error scheduling job")
		}
	}

	s.StartAsync()
}
