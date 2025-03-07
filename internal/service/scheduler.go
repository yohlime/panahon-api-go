package service

import (
	"context"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
)

func ScheduleJobs(ctx context.Context, store db.Store, conf util.Config, logger *zerolog.Logger) {
	s, err := gocron.NewScheduler()
	if err != nil {
		logger.Fatal().Err(err).Str("service", "Initialization").Msg("error scheduling job")
	}

	for _, job := range conf.CronJobs {
		var (
			jobFunc   any
			jobParams []any
			cronSched string
			jobName   string
		)

		davisFactory := func(creds sensor.DavisAPICredentials, sleepDuration time.Duration) sensor.DavisSensor {
			return sensor.NewDavis(creds, sleepDuration)
		}
		switch job.Name {
		case "lufft":
			jobName = job.Name
			cronSched = job.Schedule
			jobFunc = InsertCurrentObservations
			jobParams = []any{ctx, store, logger}
		case "davisV1":
			jobName = job.Name
			cronSched = job.Schedule
			jobFunc = InsertCurrentDavisObservations
			jobParams = []any{ctx, davisFactory, store, logger}
		case "davisV2":
			jobName = job.Name
			cronSched = job.Schedule
			jobFunc = InsertCurrentDavisObservationsV2
			jobParams = []any{ctx, davisFactory, store, logger}
		case "davisDashboard":
			jobName = job.Name
			cronSched = job.Schedule
			jobFunc = InsertCurrentDavisObservationsDashboard
			jobParams = []any{ctx, davisFactory, store, logger}
		default:
			logger.Warn().Str("service", job.Name).Msg("cron job not supported")
			continue
		}

		if _, err := s.NewJob(
			gocron.CronJob(cronSched, false),
			gocron.NewTask(jobFunc, jobParams...),
			gocron.WithName(jobName),
		); err != nil {
			logger.Fatal().Err(err).Str("service", jobName).Msg("error scheduling job")
		}
	}
	s.Start()
}
