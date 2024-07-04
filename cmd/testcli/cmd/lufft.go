package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var nLufftReqs int

var lufftCmd = &cobra.Command{
	Use:   "lufft",
	Short: "Send lufft data to the api",
	Run: func(cmd *cobra.Command, args []string) {
		sendLufft()
	},
}

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func init() {
	lufftCmd.Flags().IntVarP(&nLufftReqs, "number", "n", 100, "number of requests to send")
	lufftCmd.Flags().IntVar(&nStations, "station", 10, "number of stations to create")
}

func sendLufft() {
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
	luffts := make(chan lufftRes, nStations)
	go isServerRunning(ready)
	go sendLufftRequest(mNums, ready, luffts)

	nStoredLuffts := 0
	bar := progressbar.Default(int64(nLufftReqs), "sending requests")
	go func() {
		start := time.Now()
		for range luffts {
			nStoredLuffts++
			bar.Add(1)
		}
		if nStoredLuffts < nLufftReqs {
			nErrs := nLufftReqs - nStoredLuffts
			logger.Warn().Msgf("%d requests not sent", nErrs)
		}
		logger.Log().Dur("duration", time.Since(start)).Msg("done sending requests")
		os.Exit(0)
	}()

	err = g.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

type lufftRes struct {
	Station models.Station            `json:"station"`
	Obs     models.StationObservation `json:"observation"`
	Health  handlers.StationHealth    `json:"health"`
}

func sendLufftRequest(mNums []string, ready <-chan bool, luffts chan<- lufftRes) {
	defer close(luffts)

	logger.Info().Msg("start sending lufft requests")

	isReady := <-ready
	if !isReady {
		os.Exit(1)
	}

	nMNums := len(mNums)
	for i := 0; i < nLufftReqs; i++ {
		l := sensor.RandomLufft()
		m := mNums[i%nMNums]
		url := fmt.Sprintf("http://%s%s/ptexter", config.HTTPServerAddress, config.APIBasePath)
		payload, err := json.Marshal(map[string]string{
			"number": m,
			"msg":    l.String(23),
		})
		if err != nil {
			logger.Error().Err(err).Msg("cannot marshal json")
			continue
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			logger.Error().Err(err).Msg("cannot create request")
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			continue
		}
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			logger.Error().Err(err).Msg("cannot read response")
			continue
		}

		var obj lufftRes
		err = json.Unmarshal(data, &obj)
		if err != nil {
			logger.Error().Err(err).Msg("cannot unmarshal response")
			continue
		}

		luffts <- obj
	}
}
