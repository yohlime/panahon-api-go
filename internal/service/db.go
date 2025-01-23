package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

func InsertCurrentObservations(ctx context.Context, store db.Store, logger *zerolog.Logger) error {
	serviceName := "InsertCurrentObservations"
	obs, err := store.InsertCurrentObservations(ctx)
	if err != nil {
		logger.Error().Err(err).Str("service", serviceName).Msg("database error")
		return err
	}
	for _, o := range obs {
		statusStr := "OFFLINE"
		if time.Since(o.Timestamp.Time) < time.Hour {
			statusStr = "ONLINE"
		}
		_, err := store.UpdateStation(ctx, db.UpdateStationParams{
			ID:     o.StationID,
			Status: pgtype.Text{String: statusStr, Valid: true},
		})
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		}
	}
	logger.Info().Str("service", serviceName).Msg("insert data successful")
	return nil
}

func InsertCurrentDavisObservations(ctx context.Context, davisFactory sensor.DavisFactory, store db.Store, logger *zerolog.Logger) error {
	serviceName := "InsertCurrentDavisObservations"
	stations, err := store.ListStations(ctx, db.ListStationsParams{})
	if err != nil {
		logger.Error().Err(err).Str("service", serviceName).Msg("database error")
		return err
	}
	count := 0
	countSuccess := 0
	for _, stn := range stations {
		if stn.StationType.String == "MO" {
			if !stn.StationUrl.Valid || stn.Status.String == "INACTIVE" {
				continue
			}

			stnUrl := strings.Replace(stn.StationUrl.String, ".xml", ".json", 1)
			sleepDuration := time.Duration(util.RandomInt(1, 5)) * time.Second
			davis := davisFactory(stnUrl, sleepDuration)
			davisObs, err := davis.FetchLatest()
			if err != nil {
				logger.Error().Err(err).Str("service", serviceName).Msg("api error")
				continue
			}
			count++

			_, err = store.CreateCurrentObservation(ctx, db.CreateCurrentObservationParams{
				StationID:     stn.ID,
				Rain:          davisObs.Rain,
				Temp:          davisObs.Temp,
				Rh:            davisObs.Rh,
				Wdir:          davisObs.Wdir,
				Wspd:          davisObs.Wspd,
				Srad:          davisObs.Srad,
				Mslp:          davisObs.Mslp,
				Tn:            davisObs.Tn,
				Tx:            davisObs.Tx,
				Gust:          davisObs.Gust,
				RainAccum:     davisObs.RainAccum,
				TnTimestamp:   davisObs.TnTimestamp,
				TxTimestamp:   davisObs.TxTimestamp,
				GustTimestamp: davisObs.GustTimestamp,
				Timestamp:     davisObs.Timestamp,
			})
			if err != nil {
				logger.Error().Err(err).Str("service", serviceName).Msg("cannot create new data")
				continue
			}
			countSuccess++
			statusStr := "OFFLINE"
			if time.Since(davisObs.Timestamp.Time) < time.Hour {
				statusStr = "ONLINE"
			}

			_, err = store.UpdateStation(ctx, db.UpdateStationParams{
				ID:     stn.ID,
				Status: pgtype.Text{String: statusStr, Valid: true},
			})
			if err != nil {
				logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
			}
		}
	}
	logger.Info().Str("service", serviceName).Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).Msg("insert data successful")
	return nil
}
