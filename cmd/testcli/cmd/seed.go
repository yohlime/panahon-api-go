package cmd

import (
	"context"
	"time"

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
	go createRandomSimCard(store, ctx, nStations, simCards)
	go createRandomStation(store, ctx, simCards, stations)

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

func createRandomSimCard(store db.Store, ctx context.Context, nItems int, simCards chan<- db.SimCard) {
	defer close(simCards)
	for i := 0; i < nItems; i++ {
		arg := db.CreateSimCardParams{
			MobileNumber: util.RandomMobileNumber(),
			Type: pgtype.Text{
				String: util.RandomString(6),
				Valid:  true,
			},
		}

		res, err := store.CreateSimCard(ctx, arg)
		if err == nil {
			simCards <- res
		}
	}
}

func createRandomStation(store db.Store, ctx context.Context, simCards <-chan db.SimCard, stations chan<- db.ObservationsStation) {
	defer close(stations)
	for simCard := range simCards {
		stn := models.RandomStation()
		res, err := store.CreateStation(ctx, db.CreateStationParams{
			Name:          stn.Name,
			Lat:           util.ToFloat4(stn.Lat),
			Lon:           util.ToFloat4(stn.Lon),
			MobileNumber:  util.ToPgText(simCard.MobileNumber),
			DateInstalled: util.ToPgDate(stn.DateInstalled),
			Province:      util.ToPgText(stn.Province),
			Region:        util.ToPgText(stn.Region),
		})
		if err == nil {
			stations <- res
		}
	}
}
