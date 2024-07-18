package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/spf13/cobra"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

var lufftCmd = &cobra.Command{
	Use:   "lufft",
	Short: "Send lufft data to the api",
}

func init() {
	lufftCmd.AddCommand(lufftRandomCmd, lufftCsvCmd)
}

type lufftRes struct {
	Station models.Station            `json:"station"`
	Obs     models.StationObservation `json:"observation"`
	Health  handlers.StationHealth    `json:"health"`
}

type lufftMsg struct {
	Status int32  `json:"-"`
	Number string `json:"number"`
	Msg    string `json:"msg"`
}

type countRes struct {
	Success, Fail, Skip uint64
}

func sendLufftRequest(ready <-chan bool, msgs <-chan lufftMsg, luffts chan<- lufftRes, count *countRes) {
	defer close(luffts)

	logger.Info().Msg("start sending lufft requests")

	isReady := <-ready
	if !isReady {
		os.Exit(1)
	}

	for msg := range msgs {
		var l sensor.Lufft
		gofakeit.Struct(&l)

		if msg.Status > 0 {
			atomic.AddUint64(&count.Skip, 1)
			continue
		}

		url := fmt.Sprintf("http://%s%s/ptexter", config.HTTPServerAddress, config.APIBasePath)
		payload, err := json.Marshal(msg)
		if err != nil {
			logger.Error().Err(err).Msg("cannot marshal json")
			atomic.AddUint64(&count.Fail, 1)
			continue
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			logger.Error().Err(err).Msg("cannot create request")
			atomic.AddUint64(&count.Fail, 1)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			res.Body.Close()
			atomic.AddUint64(&count.Fail, 1)
			continue
		}

		data, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			logger.Error().Err(err).Msg("cannot read response")
			atomic.AddUint64(&count.Fail, 1)
			continue
		}

		var obj lufftRes
		err = json.Unmarshal(data, &obj)
		if err != nil {
			logger.Error().Err(err).Msg("cannot unmarshal response")
			atomic.AddUint64(&count.Fail, 1)
			continue
		}

		atomic.AddUint64(&count.Success, 1)
		luffts <- obj
	}
}
