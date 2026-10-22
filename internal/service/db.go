package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

func InsertCurrentObservations(ctx context.Context, store db.Store, logger *zerolog.Logger, onlineThreshold time.Duration) error {
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
		statusStr := stationStatus(o.Timestamp.Time, onlineThreshold)
		if statusStr == "ONLINE" {
			countOnline++
		}
		if _, err := store.SetStationStatus(ctx, db.SetStationStatusParams{
			ID:     o.StationID,
			Status: pgtype.Text{String: statusStr, Valid: true},
		}); err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		}
	}
	logger.Info().Str("service", serviceName).Str("online", fmt.Sprintf("%d/%d", countOnline, count)).Msg("insert data successful")
	return nil
}

func InsertCurrentDavisObservations(ctx context.Context, davisFactory sensor.DavisFactory, store db.Store, logger *zerolog.Logger, onlineThreshold time.Duration) error {
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
		apiToken := getQueryParamInsensitive(parsedUrl, "apiToken")
		if apiToken == "" {
			logger.Warn().
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("missing API Token")
		}
		creds := sensor.DavisAPICredentials{
			Type:     "v1",
			User:     parsedUrl.Query().Get("user"),
			Pass:     parsedUrl.Query().Get("pass"),
			APIToken: apiToken,
		}
		davis, err := davisFactory(creds, sleepDuration)
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("invalid credentials")
			continue
		}
		davisObs, err := davis.FetchLatest()
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("api error")
			continue
		}
		count++

		err = storeDavisToCurrentObservation(stn.ID, davisObs[0], ctx, store)
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("cannot create new data")
			continue
		}
		countSuccess++
		statusStr := stationStatus(davisObs[0].Timestamp.Time, onlineThreshold)

		_, err = store.SetStationStatus(ctx, db.SetStationStatusParams{
			ID:     stn.ID,
			Status: pgtype.Text{String: statusStr, Valid: true},
		})
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("update status error")
		}
	}
	logger.Info().
		Str("service", serviceName).
		Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).
		Msg("insert data successful")
	return nil
}

func InsertCurrentDavisObservationsV2(ctx context.Context, davisFactory sensor.DavisFactory, store db.Store, logger *zerolog.Logger, onlineThreshold time.Duration) error {
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
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", dStn.StationID).
				Msg("database error")
			continue
		}
		if stn.Status.String == "INACTIVE" || !dStn.APIKey.Valid || dStn.APIKey.String == "" || !dStn.APISecret.Valid || dStn.APISecret.String == "" {
			continue
		}

		sleepDuration := time.Duration(util.RandomInt(1, 5)) * time.Second

		creds := sensor.DavisAPICredentials{
			Type:      "v2",
			APIKey:    dStn.APIKey.String,
			APISecret: dStn.APISecret.String,
		}
		davis, err := davisFactory(creds, sleepDuration)
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("invalid credentials")
			continue
		}
		davisObs, err := davis.FetchLatest()
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("api error")
			continue
		}
		count++

		err = storeDavis(stn.ID, davisObs[0], ctx, store)
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("cannot create new data")
			continue
		}
		countSuccess++
		statusStr := stationStatus(davisObs[0].Timestamp.Time, onlineThreshold)
		if _, err := store.SetStationStatus(ctx, db.SetStationStatusParams{
			ID:     stn.ID,
			Status: pgtype.Text{String: statusStr, Valid: true},
		}); err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		}
	}
	logger.Info().Str("service", serviceName).Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).Msg("insert data successful")
	return nil
}

func InsertCurrentDavisObservationsDashboard(ctx context.Context, davisFactory sensor.DavisFactory, store db.Store, logger *zerolog.Logger, onlineThreshold time.Duration) error {
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
			logger.Error().
				Err(err).Str("service", serviceName).
				Int64("station_id", dStn.StationID).
				Msg("database error")
			continue
		}
		if stn.Status.String == "INACTIVE" || !dStn.Uuid.Valid || dStn.Uuid.String == "" {
			continue
		}

		sleepDuration := time.Duration(util.RandomInt(1, 3)) * time.Second

		creds := sensor.DavisAPICredentials{
			Type:    "dashboard",
			StnUUID: dStn.Uuid.String,
		}
		davis, err := davisFactory(creds, sleepDuration)
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("invalid credentials")
			continue
		}
		davisObs, err := davis.FetchLatest()
		if err != nil {
			logger.Error().
				Err(err).Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("api error")
			continue
		}
		count++
		// logger.Debug().Interface("davis", davisObs).Str("service", serviceName)

		err = storeDavis(stn.ID, davisObs[0], ctx, store)
		if err != nil {
			logger.Error().Err(err).
				Str("service", serviceName).
				Int64("station_id", stn.ID).
				Msg("cannot create new data")
			continue
		}
		countSuccess++
		statusStr := stationStatus(davisObs[0].Timestamp.Time, onlineThreshold)
		if _, err := store.SetStationStatus(ctx, db.SetStationStatusParams{
			ID:     stn.ID,
			Status: pgtype.Text{String: statusStr, Valid: true},
		}); err != nil {
			logger.Error().Err(err).Str("service", serviceName).Msg("update status error")
		}
	}
	logger.Info().
		Str("service", serviceName).
		Str("success", fmt.Sprintf("%d/%d", countSuccess, count)).
		Msg("insert data successful")
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

func getQueryParamInsensitive(u *url.URL, key string) string {
	params := u.Query()
	for k, vals := range params {
		if strings.EqualFold(k, key) && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

// stationStatus returns the online state for an observation timestamp.
func stationStatus(ts time.Time, onlineThreshold time.Duration) string {
	if time.Since(ts) < onlineThreshold {
		return "ONLINE"
	}
	return "OFFLINE"
}

// RefreshStationStatus marks ONLINE stations with no fresh observation OFFLINE.
func RefreshStationStatus(ctx context.Context, store db.Store, logger *zerolog.Logger, onlineThreshold time.Duration) error {
	serviceName := "RefreshStationStatus"
	n, err := store.MarkStaleStationsOffline(ctx, onlineThreshold.Seconds())
	if err != nil {
		logger.Error().Err(err).Str("service", serviceName).Msg("database error")
		return err
	}
	logger.Info().Str("service", serviceName).Int64("offline", n).Msg("station status refreshed")
	return nil
}
