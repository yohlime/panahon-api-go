package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var nLufftReqs int

var lufftRandomCmd = &cobra.Command{
	Use:   "rnd",
	Short: "Send random lufft data to the api",
	Run: func(cmd *cobra.Command, args []string) {
		sendRandomLufft()
	},
}

func init() {
	lufftRandomCmd.Flags().IntVarP(&nLufftReqs, "requests", "n", 100, "number of requests to send")
	lufftRandomCmd.Flags().IntVar(&nStations, "station", 10, "number of stations to create")
}

func sendRandomLufft() {
	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, store := dbConnect(ctx)
	defer connPool.Close()

	n32Stns := int32(nStations)
	stns, err := store.ListStations(ctx, db.ListStationsParams{Limit: util.ToInt4(&n32Stns)})
	if err != nil {
		logger.Fatal().Err(err).Msg("error getting stations")
	}

	n := len(stns)
	if n < nStations {
		msg := ""
		if n == 0 {
			msg = "no stations found"
		} else {
			msg = fmt.Sprintf("only %d stations found", n)
		}
		dStns := nStations - n
		logger.Info().Msgf("%s, creating %d stations", msg, dStns)
		_nStations := nStations
		nStations = dStns
		stns = append(stns, seedStations()...)
		nStations = _nStations
	}

	var mNums []string
	for _, stn := range stns {
		if stn.MobileNumber.Valid {
			mNums = append(mNums, stn.MobileNumber.String)
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	runGinServer(ctx, g, store)

	ready := make(chan bool)
	msgs := make(chan lufftMsg, nStations)
	luffts := make(chan lufftRes, 12)
	count := &countRes{}
	go isServerRunning(ready)
	go generateLufftMsg(mNums, msgs)
	go sendLufftRequest(ready, msgs, luffts, count)

	bar := progressbar.Default(int64(nLufftReqs), "sending requests")
	start := time.Now()
	for range luffts {
		bar.Add(1)
	}

	logger.Log().
		Dur("duration", time.Since(start)).
		Int("success", int(count.Success)).
		Int("fail", int(count.Fail)).
		Int("skip", int(count.Skip)).
		Msg("done sending requests")
	os.Exit(0)

	err = g.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

func generateLufftMsg(mNums []string, msgs chan<- lufftMsg) {
	defer close(msgs)

	nMNums := len(mNums)

	for i := 0; i < nLufftReqs; i++ {
		var l sensor.Lufft
		gofakeit.Struct(&l)

		m := mNums[i%nMNums]

		msg := lufftMsg{
			Number: m,
			Msg:    l.String(23),
		}

		msgs <- msg
	}
}
