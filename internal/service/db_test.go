package service

import (
	"context"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocksensor "github.com/emiliogozo/panahon-api-go/internal/mocks/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInsertCurrentDavisObservations(t *testing.T) {
	testCases := []struct {
		name          string
		buildStubs    func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
		checkResponse func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore)
	}{
		{
			name: "default",
			buildStubs: func(davisSensor *mocksensor.MockDavisSensor, store *mockdb.MockStore) {
				stations := []db.ObservationsStation{
					{
						Name: "stn001",
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
					},
					{
						Name: "stn002",
						StationType: pgtype.Text{
							String: "MO",
							Valid:  true,
						},
						StationUrl: pgtype.Text{
							String: "http://api.com/current/a.json",
							Valid:  false,
						},
						Status: pgtype.Text{
							String: "ONLINE",
							Valid:  true,
						},
					},
					{
						Name: "stn003",
						StationType: pgtype.Text{
							String: "MO",
							Valid:  true,
						},
						StationUrl: pgtype.Text{
							String: "http://api.com/current/a.json",
							Valid:  true,
						},
						Status: pgtype.Text{
							String: "INACTIVE",
							Valid:  true,
						},
					},
					{
						Name: "stn004",
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
					},
				}

				davisObs := []*sensor.DavisCurrentObservation{
					{
						Rain: pgtype.Float4{
							Float32: 5.0,
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: 28.6,
							Valid:   true,
						},
						Timestamp: pgtype.Timestamptz{
							Time:  time.Now(),
							Valid: true,
						},
					},
					{
						Rain: pgtype.Float4{
							Float32: 5.0,
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: 28.6,
							Valid:   true,
						},
					},
					{
						Rain: pgtype.Float4{
							Float32: 5.0,
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: 28.6,
							Valid:   true,
						},
					},
					{
						Rain: pgtype.Float4{
							Float32: 5.0,
							Valid:   true,
						},
						Temp: pgtype.Float4{
							Float32: 28.6,
							Valid:   true,
						},
						Timestamp: pgtype.Timestamptz{
							Time:  time.Now().Add(-2 * time.Hour),
							Valid: true,
						},
					},
				}

				stnStatus := []string{"ONLINE", "OFFLINE", "OFFLINE", "OFFLINE"}

				store.EXPECT().ListStations(mock.AnythingOfType("backgroundCtx"), mock.AnythingOfType("db.ListStationsParams")).
					Return(stations, nil)

				for i, stn := range stations {
					if stn.StationType.String == "MO" {
						if !stn.StationUrl.Valid || stn.Status.String == "INACTIVE" {
							continue
						}
						davisSensor.EXPECT().FetchLatest().Return(davisObs[i], nil).Once()
						store.EXPECT().CreateCurrentObservation(mock.AnythingOfType("backgroundCtx"), mock.MatchedBy(func(arg db.CreateCurrentObservationParams) bool {
							return assert.InDelta(t, arg.Rain.Float32, davisObs[i].Rain.Float32, 0.01)
						})).
							Return(db.ObservationsCurrent{}, nil).Once()
						store.EXPECT().UpdateStation(mock.AnythingOfType("backgroundCtx"), mock.MatchedBy(func(arg db.UpdateStationParams) bool {
							return arg.Status.String == stnStatus[i]
						})).
							Return(db.ObservationsStation{}, nil).Once()
					}
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

			sensorFactory := func(sensorUrl string, sleepDuration time.Duration) sensor.DavisSensor {
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
