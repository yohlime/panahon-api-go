package service

import (
	"context"
	"time"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog"
)

func ScheduleJobs(ctx context.Context, store db.Store, logger *zerolog.Logger) {
	s := gocron.NewScheduler(time.Local)

	_, err := s.Cron("2-59/10 * * * *").Tag("InsertCurrentObservations").Do(InsertCurrentObservations, ctx, store, logger)
	if err != nil {
		logger.Fatal().Err(err).Str("service", "InsertCurrentObservations").Msg("error scheduling job")
	}

	_, err = s.Cron("*/10 * * * *").Tag("InsertCurrentDavisObservations").Do(InsertCurrentDavisObservations, ctx, store, logger)
	if err != nil {
		logger.Fatal().Err(err).Str("service", "InsertCurrentDavisObservations").Msg("error scheduling job")
	}

	s.StartAsync()
}
