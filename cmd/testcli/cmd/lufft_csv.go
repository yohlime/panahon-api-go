package cmd

import (
	"context"
	"encoding/csv"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var lufftCsvCmd = &cobra.Command{
	Use:   "csv",
	Short: "Send lufft data from csv to the api",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		processCsvLufft(filePath)
	},
}

func init() {
}

func processCsvLufft(filePath string) {
	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, store := dbConnect(ctx)
	defer connPool.Close()

	g, ctx := errgroup.WithContext(ctx)
	runGinServer(ctx, g, store)

	ready := make(chan bool)
	msgs := make(chan lufftMsg, 12)
	luffts := make(chan lufftRes, 12)
	count := &countRes{}
	go isServerRunning(ready)
	go generateLuffMsgFromCsv(ctx, filePath, store, msgs)
	go sendLufftRequest(ready, msgs, luffts, count)

	bar := progressbar.Default(-1, "sending requests")
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

	err := g.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

func generateLuffMsgFromCsv(ctx context.Context, filePath string, store db.Store, msgs chan<- lufftMsg) {
	defer close(msgs)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Fatal().Err(err)
		}
		if len(record) != 3 {
			continue
		}

		status, err := strconv.Atoi(record[0])
		if err != nil {
			log.Fatal().Err(err)
		}
		msg := lufftMsg{
			Status: int32(status),
			Number: record[1],
			Msg:    record[2],
		}

		_, err = store.GetStationByMobileNumber(ctx,
			pgtype.Text{
				String: msg.Number,
				Valid:  msg.Number != "",
			},
		)
		if err != nil {
			if errors.Is(err, db.ErrRecordNotFound) {
				sim := generateRandomSimCard(store, ctx, msg.Number)
				if sim == nil {
					continue
				}
				stn := generateRandomStation(store, ctx, *sim)
				if stn == nil {
					continue
				}
				msgs <- msg
			}
			continue
		}
		msgs <- msg
	}
}
