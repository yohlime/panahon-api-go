package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateMisolStationTxParams struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Lat       pgtype.Float4 `json:"lat"`
	Lon       pgtype.Float4 `json:"lon"`
	Elevation pgtype.Float4 `json:"elevation"`
	Province  pgtype.Text   `json:"province"`
	Region    pgtype.Text   `json:"region"`
	Address   pgtype.Text   `json:"address"`
}

type CreateMisolStationTxResult struct {
	ID   int64
	Info ObservationsStation
}

func (store *SQLStore) CreateMisolStationTx(ctx context.Context, arg CreateMisolStationTxParams) (CreateMisolStationTxResult, error) {
	var result CreateMisolStationTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Info, err = q.CreateStation(ctx, CreateStationParams{
			Name:      arg.Name,
			Lat:       arg.Lat,
			Lon:       arg.Lon,
			Elevation: arg.Elevation,
			Province:  arg.Province,
			Region:    arg.Region,
			Address:   arg.Address,
			StationType: pgtype.Text{
				String: "MISOL",
				Valid:  true,
			},
		})
		if err != nil {
			return err
		}

		mStn, err := q.CreateMisolStation(ctx, CreateMisolStationParams{
			ID:        arg.ID,
			StationID: result.Info.ID,
		})
		if err != nil {
			return err
		}
		result.ID = mStn.ID

		return nil
	})

	return result, err
}
