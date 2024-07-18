package cmd

import (
	"context"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var nStations int

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seeds the database with test data",
}

var seedStationsCmd = &cobra.Command{
	Use:   "stations",
	Short: "Seeds the stations table with test data",
	Run: func(cmd *cobra.Command, args []string) {
		seedStations()
	},
}

func init() {
	seedCmd.AddCommand(seedStationsCmd)
	seedStationsCmd.Flags().IntVar(&nStations, "station", 10, "number of stations to create")
}

func seedStations() []db.ObservationsStation {
	ctx := context.Background()

	connPool, store := dbConnect(ctx)
	defer connPool.Close()

	simCards := make(chan db.SimCard, 5)
	stations := make(chan db.ObservationsStation, 5)

	logger.Info().Msgf("adding %d stations...\n", nStations)
	go storeRandomSimCard(store, ctx, nStations, simCards)
	go storeRandomStation(store, ctx, simCards, stations)

	var stns []db.ObservationsStation
	bar := progressbar.Default(int64(nStations), "creating stations")
	start := time.Now()
	for stn := range stations {
		stns = append(stns, stn)
		bar.Add(1)
	}
	nStoredStations := len(stns)

	if nStoredStations < nStations {
		nErr := nStations - nStoredStations
		logger.Warn().Msgf("failed to add %d stations", nErr)
	}
	logger.Log().Dur("duration", time.Since(start)).Msg("done creating stations")

	return stns
}

func generateRandomSimCard(store db.Store, ctx context.Context, mobileNumber string) *db.SimCard {
	if mobileNumber == "" {
		mobileNumber = gofakeit.Regex("639[0-9]{9}")
	}
	arg := db.CreateSimCardParams{
		MobileNumber: mobileNumber,
		Type: pgtype.Text{
			String: gofakeit.RandomString([]string{"globe", "smart"}),
			Valid:  true,
		},
	}

	res, err := store.CreateSimCard(ctx, arg)
	if err != nil {
		return nil
	}

	return &res
}

func storeRandomSimCard(store db.Store, ctx context.Context, nItems int, simCards chan<- db.SimCard) {
	defer close(simCards)
	for i := 0; i < nItems; i++ {
		simCard := generateRandomSimCard(store, ctx, "")
		if simCard != nil {
			simCards <- *simCard
		}
	}
}

func generateRandomStation(store db.Store, ctx context.Context, simCard db.SimCard) *db.ObservationsStation {
	var s models.Station
	gofakeit.Struct(&s)

	stn, err := store.CreateStation(ctx, db.CreateStationParams{
		Name:         s.Name,
		Lat:          util.ToFloat4(s.Lat),
		Lon:          util.ToFloat4(s.Lon),
		MobileNumber: util.ToPgText(simCard.MobileNumber),
		DateInstalled: pgtype.Date{
			Time:  s.DateInstalled.Time,
			Valid: !s.DateInstalled.IsZero(),
		},
		Province: util.ToPgText(string(s.Province)),
		Region:   util.ToPgText(string(s.Region)),
	})
	if err != nil {
		return nil
	}
	return &stn
}

func storeRandomStation(store db.Store, ctx context.Context, simCards <-chan db.SimCard, stations chan<- db.ObservationsStation) {
	defer close(stations)
	for simCard := range simCards {
		stn := generateRandomStation(store, ctx, simCard)
		if stn != nil {
			stations <- *stn
		}
	}
}
