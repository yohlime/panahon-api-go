package service

import (
	"context"
	"fmt"
	"net/url"
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
	obs1, err := store.InsertCurrentMOObservations(ctx)
	if err != nil {
		logger.Error().Err(err).Str("service", serviceName).Msg("database error")
		return err
	}
	obs = append(obs, obs1...)
	count := len(obs)
	countOnline := 0
	for _, o := range obs {
		statusStr := "OFFLINE"
		if time.Since(o.Timestamp.Time) < time.Hour {
			statusStr = "ONLINE"
			countOnline++
		}
		if _, err := store.UpdateStation(ctx, db.UpdateStationParams{
			ID:     o.StationID,
			Status: pgtype.Text{String: statusStr, Valid: true},
		}); err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		}
	}
	logger.Info().Str("service", serviceName).Str("online", fmt.Sprintf("%d/%d", countOnline, count)).Msg("insert data successful")
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
		if stn.StationType.String != "MO" || !stn.StationUrl.Valid || stn.Status.String == "INACTIVE" {
			continue
		}

		sleepDuration := time.Duration(util.RandomInt(1, 5)) * time.Second
		parsedUrl, err := url.Parse(stn.StationUrl.String)
		if err != nil {
			continue
		}
		creds := sensor.DavisAPICredentials{
			User:     parsedUrl.Query().Get("user"),
			Pass:     parsedUrl.Query().Get("pass"),
			APIToken: parsedUrl.Query().Get("apiToken"),
		}
		davis := davisFactory(creds, sleepDuration)
		davisObs, err := davis.FetchLatest()
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("api error")
			continue
		}
		count++

		err = storeDavisToCurrentObservation(stn.ID, davisObs[0], ctx, store)
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("cannot create new data")
			continue
		}
		countSuccess++
		statusStr := "OFFLINE"
		if time.Since(davisObs[0].Timestamp.Time) < time.Hour {
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
	logger.Info().Str("service", serviceName).Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).Msg("insert data successful")
	return nil
}

func InsertCurrentDavisObservationsV2(ctx context.Context, davisFactory sensor.DavisFactory, store db.Store, logger *zerolog.Logger) error {
	serviceName := "InsertCurrentDavisObservationsV2"
	stations, err := store.ListWeatherlinkStations(ctx, db.ListWeatherlinkStationsParams{})
	if err != nil {
		logger.Error().Err(err).Str("service", serviceName).Msg("database error")
		return err
	}
	count := 0
	countSuccess := 0
	for _, dStn := range stations {
		stn, err := store.GetStation(ctx, dStn.StationID)
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("database error")
			continue
		}
		if stn.Status.String == "INACTIVE" || !dStn.ApiKey.Valid || dStn.ApiKey.String == "" || !dStn.ApiSecret.Valid || dStn.ApiSecret.String == "" {
			// logger.Info().Str("service", serviceName).Str("station", stn.Name).Msg("inactive or invalid api credentials")
			continue
		}

		sleepDuration := time.Duration(util.RandomInt(1, 5)) * time.Second

		creds := sensor.DavisAPICredentials{
			APIKey:    dStn.ApiKey.String,
			APISecret: dStn.ApiSecret.String,
		}
		davis := davisFactory(creds, sleepDuration)
		davisObs, err := davis.FetchLatest()
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("api error")
			continue
		}
		count++

		err = storeDavis(stn.ID, davisObs[0], ctx, store)
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("cannot create new data")
			continue
		}
		countSuccess++
		// statusStr := "OFFLINE"
		// if time.Since(davisObs[0].Timestamp.Time) < time.Hour {
		// 	statusStr = "ONLINE"
		// }
		//
		// _, err = store.UpdateStation(ctx, db.UpdateStationParams{
		// 	ID:     stn.ID,
		// 	Status: pgtype.Text{String: statusStr, Valid: true},
		// })
		// if err != nil {
		// 	logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		// }
	}
	logger.Info().Str("service", serviceName).Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).Msg("insert data successful")
	return nil
}

func InsertCurrentDavisObservationsDashboard(ctx context.Context, davisFactory sensor.DavisFactory, store db.Store, logger *zerolog.Logger) error {
	serviceName := "InsertCurrentDavisObservationsDashboard"
	stations, err := store.ListWeatherlinkStations(ctx, db.ListWeatherlinkStationsParams{})
	if err != nil {
		logger.Error().Err(err).Str("service", serviceName).Msg("database error")
		return err
	}
	count := 0
	countSuccess := 0
	for _, dStn := range stations {
		stn, err := store.GetStation(ctx, dStn.StationID)
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("database error")
			continue
		}
		if stn.Status.String == "INACTIVE" || !dStn.Uuid.Valid || dStn.Uuid.String == "" {
			// logger.Info().Str("service", serviceName).Str("station", stn.Name).Msg("inactive or missing station uuid")
			continue
		}

		sleepDuration := time.Duration(util.RandomInt(1, 3)) * time.Second

		creds := sensor.DavisAPICredentials{
			StnUUID: dStn.Uuid.String,
		}
		davis := davisFactory(creds, sleepDuration)
		davisObs, err := davis.FetchLatest()
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("api error")
			continue
		}
		count++
		logger.Debug().Interface("davis", davisObs).Str("service", serviceName)

		err = storeDavis(stn.ID, davisObs[0], ctx, store)
		if err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("cannot create new data")
			continue
		}
		countSuccess++
		// statusStr := "OFFLINE"
		// if time.Since(davisObs[0].Timestamp.Time) < time.Hour {
		// 	statusStr = "ONLINE"
		// }
		//
		// _, err = store.UpdateStation(ctx, db.UpdateStationParams{
		// 	ID:     stn.ID,
		// 	Status: pgtype.Text{String: statusStr, Valid: true},
		// })
		// if err != nil {
		// 	logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		// }
	}
	logger.Info().Str("service", serviceName).Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).Msg("insert data successful")
	return nil
}

func storeDavis(stnID int64, o sensor.DavisCurrentObservation, ctx context.Context, store db.Store) error {
	_, err := store.CreateStationMOObservation(ctx, db.CreateStationMOObservationParams{
		StationID: stnID,
		Rr:        o.Rr,
		Temp:      o.Temp,
		Rh:        o.Rh,
		Wdir:      o.Wdir,
		Wspd:      o.Wspd,
		Wspdx:     o.Wspdx,
		Srad:      o.Srad,
		Pres:      o.Pres,
		Hi:        o.Hi,
		QcLevel:   0,
		Timestamp: o.Timestamp,
	})
	return err
}

func storeDavisToCurrentObservation(stnID int64, o sensor.DavisCurrentObservation, ctx context.Context, store db.Store) error {
	_, err := store.CreateCurrentObservation(ctx, db.CreateCurrentObservationParams{
		StationID:     stnID,
		Rain:          o.Rr,
		Temp:          o.Temp,
		Rh:            o.Rh,
		Wdir:          o.Wdir,
		Wspd:          o.Wspd,
		Srad:          o.Srad,
		Mslp:          o.Pres,
		Tn:            o.Tn,
		Tx:            o.Tx,
		Gust:          o.Wspdx,
		RainAccum:     o.RainAccum,
		TnTimestamp:   o.TnTimestamp,
		TxTimestamp:   o.TxTimestamp,
		GustTimestamp: o.GustTimestamp,
		Timestamp:     o.Timestamp,
	})
	return err
}
