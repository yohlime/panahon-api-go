package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocksensor "github.com/emiliogozo/panahon-api-go/internal/mocks/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInsertCurrentDavisObservations(t *testing.T) {
	testCases := []struct {
		name          string
		buildStubs    func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
		checkResponse func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
	}{
		{
			name: "Default",
			buildStubs: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				stns := make([]db.ObservationsStation, 4)
				davisObsSlice := make([]sensor.DavisCurrentObservation, 0)
				stnStatus := make([]string, 0)
				for i := range stns {
					stn := db.ObservationsStation{
						Name: fmt.Sprintf("stn%03d", i),
						StationType: pgtype.Text{
							String: "MO",
							Valid:  true,
						},
						StationUrl: pgtype.Text{
							String: "http://api.com/current/a.json",
							Valid:  true,
						},
						Status: pgtype.Text{
							String: "ONLINE",
							Valid:  true,
						},
					}
					if i == 1 {
						stn.StationUrl.Valid = false
					}
					if i == 2 {
						stn.Status.String = "INACTIVE"
					}
					stns[i] = stn

					dObs := sensor.DavisCurrentObservation{
						Rr: pgtype.Float4{
							Float32: util.RandomFloat[float32](2.5, 10.6),
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: util.RandomFloat[float32](20.5, 33.7),
							Valid:   true,
						},
					}
					if i == 0 {
						dObs.Timestamp = pgtype.Timestamptz{
							Time:  time.Now(),
							Valid: true,
						}
						stnStatus = append(stnStatus, "ONLINE")
					} else {
						if i == 3 {
							dObs.Timestamp = pgtype.Timestamptz{
								Time:  time.Now().Add(-2 * time.Hour),
								Valid: true,
							}
						}
						stnStatus = append(stnStatus, "OFFLINE")
					}
					davisObsSlice = append(davisObsSlice, dObs)
				}

				store.EXPECT().ListStations(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.ListStationsParams")).
					Return(stns, nil)

				for i, stn := range stns {
					dObs := davisObsSlice[i]
					if stn.StationType.String != "MO" || !stn.StationUrl.Valid || stn.Status.String == "INACTIVE" {
						continue
					}
					davisSensor.EXPECT().FetchLatest().Return([]sensor.DavisCurrentObservation{dObs}, nil).Once()
					store.EXPECT().CreateCurrentObservation(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.CreateCurrentObservationParams")).
						Run(func(ctx context.Context, arg db.CreateCurrentObservationParams) {
							require.InDelta(t, dObs.Rr.Float32, arg.Rain.Float32, 0.01)
						}).
						Return(db.ObservationsCurrent{}, nil).Once()
					store.EXPECT().UpdateStation(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.UpdateStationParams")).
						Run(func(ctx context.Context, arg db.UpdateStationParams) {
							require.Equal(t, stnStatus[i], arg.Status.String)
						}).
						Return(db.ObservationsStation{}, nil).Once()
				}
			},
			checkResponse: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				davisSensor.AssertExpectations(t)
				store.AssertExpectations(t)
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			davisSensor := mocksensor.NewMockDavisSensor(t)
			tc.buildStubs(davisSensor, store)

			sensorFactory := func(cred sensor.DavisAPICredentials, sleepDuration time.Duration) sensor.DavisSensor {
				return davisSensor
			}

			config := util.Config{
				EnableFileLogging: false,
			}
			logger := util.NewLogger(config)

			ctx := context.Background()
			InsertCurrentDavisObservations(ctx, sensorFactory, store, logger)
			tc.checkResponse(davisSensor, store)
		})
	}
}

func TestInsertCurrentDavisObservationsV2(t *testing.T) {
	testCases := []struct {
		name          string
		buildStubs    func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
		checkResponse func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
	}{
		{
			name: "Default",
			buildStubs: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				stns := make([]db.ObservationsStation, 0)
				davisStns := make([]db.Weatherlink, 0)
				davisObsSlice := make([]sensor.DavisCurrentObservation, 0)
				stnStatus := make([]string, 0)
				for i, stnID := range []int64{24, 37, 555, 1117, 1317} {
					stn := db.ObservationsStation{
						ID:   stnID,
						Name: fmt.Sprintf("stn%5d", stnID),
						StationType: pgtype.Text{
							String: "MO",
							Valid:  true,
						},
						Status: pgtype.Text{
							String: "ONLINE",
							Valid:  true,
						},
					}
					if i == 2 {
						stn.Status.String = "INACTIVE"
					}
					stns = append(stns, stn)

					dStn := db.Weatherlink{
						StationID: stnID,
						ApiKey: pgtype.Text{
							String: util.RandomString(12),
							Valid:  true,
						},
					}
					if i != 1 {
						dStn.ApiSecret = pgtype.Text{
							String: util.RandomString(24),
							Valid:  true,
						}
					} else {
						dStn.Uuid = pgtype.Text{
							String: util.RandomString(32),
							Valid:  true,
						}
					}
					davisStns = append(davisStns, dStn)

					dObs := sensor.DavisCurrentObservation{
						Rr: pgtype.Float4{
							Float32: util.RandomFloat[float32](2.0, 12.7),
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: util.RandomFloat[float32](22.0, 32.9),
							Valid:   true,
						},
					}
					if i == 0 || i == 3 {
						dObs.Timestamp = pgtype.Timestamptz{
							Time:  time.Now(),
							Valid: true,
						}
						stnStatus = append(stnStatus, "ONLINE")
					} else {
						if i == 4 {
							dObs.Timestamp = pgtype.Timestamptz{
								Time:  time.Now().Add(-2 * time.Hour),
								Valid: true,
							}
						}
						stnStatus = append(stnStatus, "OFFLINE")
					}
					davisObsSlice = append(davisObsSlice, dObs)
				}

				store.EXPECT().ListWeatherlinkStations(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.ListWeatherlinkStationsParams")).
					Return(davisStns, nil)

				for i, dStn := range davisStns {
					stn := stns[i]
					dObs := davisObsSlice[i]
					fmt.Printf("stn%05d\n", stn.ID)
					store.EXPECT().GetStation(mock.AnythingOfType("backgroundCtx"), dStn.StationID).Return(stn, nil).Once()
					if stn.Status.String == "INACTIVE" || !((dStn.ApiKey.Valid && dStn.ApiKey.String != "") && (dStn.ApiSecret.Valid && dStn.ApiSecret.String != "")) {
						continue
					}
					davisSensor.EXPECT().FetchLatest().Return([]sensor.DavisCurrentObservation{dObs}, nil).Once()
					store.EXPECT().CreateStationMOObservation(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.CreateStationMOObservationParams")).
						Run(func(ctx context.Context, arg db.CreateStationMOObservationParams) {
							require.InDelta(t, dObs.Rr.Float32, arg.Rr.Float32, 0.01)
						}).
						Return(db.ObservationsMoObservation{}, nil).Once()
					// store.EXPECT().UpdateStation(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.UpdateStationParams")).
					// 	Run(func(ctx context.Context, arg db.UpdateStationParams) {
					// 		require.Equal(t, stnStatus[i], arg.Status.String)
					// 	}).
					// 	Return(db.ObservationsStation{}, nil).Once()
				}
			},
			checkResponse: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				davisSensor.AssertExpectations(t)
				store.AssertExpectations(t)
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			davisSensor := mocksensor.NewMockDavisSensor(t)
			tc.buildStubs(davisSensor, store)

			sensorFactory := func(cred sensor.DavisAPICredentials, sleepDuration time.Duration) sensor.DavisSensor {
				return davisSensor
			}

			config := util.Config{
				EnableFileLogging: false,
			}
			logger := util.NewLogger(config)

			ctx := context.Background()
			InsertCurrentDavisObservationsV2(ctx, sensorFactory, store, logger)
			tc.checkResponse(davisSensor, store)
		})
	}
}

func TestInsertCurrentDavisObservationsDashboard(t *testing.T) {
	testCases := []struct {
		name          string
		buildStubs    func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
		checkResponse func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
	}{
		{
			name: "Default",
			buildStubs: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				stns := make([]db.ObservationsStation, 0)
				davisStns := make([]db.Weatherlink, 0)
				davisObsSlice := make([]sensor.DavisCurrentObservation, 0)
				stnStatus := make([]string, 0)
				for i, stnID := range []int64{24, 37, 555, 1117, 1317} {
					stn := db.ObservationsStation{
						ID:   stnID,
						Name: fmt.Sprintf("stn%05d", stnID),
						StationType: pgtype.Text{
							String: "MO",
							Valid:  true,
						},
						Status: pgtype.Text{
							String: "ONLINE",
							Valid:  true,
						},
					}
					if i == 2 {
						stn.Status.String = "INACTIVE"
					}
					stns = append(stns, stn)

					dStn := db.Weatherlink{
						StationID: stnID,
					}
					if i != 1 {
						dStn.Uuid = pgtype.Text{
							String: util.RandomString(36),
							Valid:  true,
						}
					}
					davisStns = append(davisStns, dStn)

					dObs := sensor.DavisCurrentObservation{
						Rr: pgtype.Float4{
							Float32: util.RandomFloat[float32](2.0, 12.7),
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: util.RandomFloat[float32](22.0, 32.9),
							Valid:   true,
						},
					}
					if i == 0 || i == 3 {
						dObs.Timestamp = pgtype.Timestamptz{
							Time:  time.Now(),
							Valid: true,
						}
						stnStatus = append(stnStatus, "ONLINE")
					} else {
						if i == 4 {
							dObs.Timestamp = pgtype.Timestamptz{
								Time:  time.Now().Add(-2 * time.Hour),
								Valid: true,
							}
						}
						stnStatus = append(stnStatus, "OFFLINE")
					}
					davisObsSlice = append(davisObsSlice, dObs)
				}

				store.EXPECT().ListWeatherlinkStations(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.ListWeatherlinkStationsParams")).
					Return(davisStns, nil)

				for i, dStn := range davisStns {
					stn := stns[i]
					dObs := davisObsSlice[i]
					store.EXPECT().GetStation(mock.AnythingOfType("backgroundCtx"), dStn.StationID).Return(stn, nil).Once()
					if stn.Status.String == "INACTIVE" || !dStn.Uuid.Valid || dStn.Uuid.String == "" {
						continue
					}
					davisSensor.EXPECT().FetchLatest().Return([]sensor.DavisCurrentObservation{dObs}, nil).Once()
					store.EXPECT().CreateStationMOObservation(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.CreateStationMOObservationParams")).
						Run(func(ctx context.Context, arg db.CreateStationMOObservationParams) {
							require.InDelta(t, dObs.Rr.Float32, arg.Rr.Float32, 0.01)
						}).
						Return(db.ObservationsMoObservation{}, nil).Once()
					// store.EXPECT().UpdateStation(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.UpdateStationParams")).
					// 	Run(func(ctx context.Context, arg db.UpdateStationParams) {
					// 		require.Equal(t, stnStatus[i], arg.Status.String)
					// 	}).
					// 	Return(db.ObservationsStation{}, nil).Once()
				}
			},
			checkResponse: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				davisSensor.AssertExpectations(t)
				store.AssertExpectations(t)
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			davisSensor := mocksensor.NewMockDavisSensor(t)
			tc.buildStubs(davisSensor, store)

			sensorFactory := func(cred sensor.DavisAPICredentials, sleepDuration time.Duration) sensor.DavisSensor {
				return davisSensor
			}

			config := util.Config{
				EnableFileLogging: false,
			}
			logger := util.NewLogger(config)

			ctx := context.Background()
			InsertCurrentDavisObservationsDashboard(ctx, sensorFactory, store, logger)
			tc.checkResponse(davisSensor, store)
		})
	}
}
