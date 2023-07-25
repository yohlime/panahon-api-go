package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/rs/zerolog/log"
)

type FirstOrCreateSimAccessTokenTxParams struct {
	MobileNumber     string          `json:"mobile_number"`
	MobileNumberType util.NullString `json:"mobile_number_type"`
	AccessToken      string          `json:"access_token"`
	AccessTokenType  string          `json:"access_token_type"`
}

type FirstOrCreateSimAccessTokenTxResult struct {
	AccessToken SimAccessToken
	IsCreated   bool
}

func (store *SQLStore) FirstOrCreateSimAccessTokenTx(ctx context.Context, arg FirstOrCreateSimAccessTokenTxParams) (FirstOrCreateSimAccessTokenTxResult, error) {
	var result FirstOrCreateSimAccessTokenTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		if arg.MobileNumber == "" {
			return fmt.Errorf("mobile number cannot be empty")
		}

		if arg.AccessTokenType == "" {
			return fmt.Errorf("type cannot be empty")
		}

		_, err = q.GetSimCard(ctx, arg.MobileNumber)
		if err != nil {
			if !errors.Is(err, ErrRecordNotFound) {
				log.Error().Err(err).
					Str("mobile_number", arg.MobileNumber).
					Msg("[SimCard] cannot get number from the database")
				return err
			}
			log.Debug().
				Str("mobile_number", arg.MobileNumber).
				Msg("[SimCard] new number detected")
			simCardArg := CreateSimCardParams{
				MobileNumber: arg.MobileNumber,
				Type:         arg.MobileNumberType,
			}
			_, err = q.CreateSimCard(ctx, simCardArg)
			if err != nil {
				log.Error().Err(err).
					Str("mobile_number", arg.MobileNumber).
					Msg("[SimCard] failed to add number")
				return err
			}
			log.Debug().
				Str("mobile_number", arg.MobileNumber).
				Msg("[SimCard] number added succesfully")
		}

		result.AccessToken, err = q.GetSimAccessToken(ctx, arg.AccessToken)
		if err != nil {
			if errors.Is(err, ErrRecordNotFound) {
				log.Debug().
					Str("mobile_number", arg.MobileNumber).
					Msg("[SimAccessToken] new token detected")
				accTokArg := CreateSimAccessTokenParams{
					AccessToken:  arg.AccessToken,
					Type:         arg.AccessTokenType,
					MobileNumber: arg.MobileNumber,
				}
				result.AccessToken, err = q.CreateSimAccessToken(ctx, accTokArg)
				if err != nil {
					log.Error().Err(err).
						Str("mobile_number", arg.MobileNumber).
						Msg("[SimAccessToken] failed to add token")
					return err
				}
				log.Debug().
					Str("mobile_number", arg.MobileNumber).
					Msg("[SimAccessToken] token added succesfully")
				result.IsCreated = true
				return nil
			}
			log.Error().Err(err).
				Str("mobile_number", arg.MobileNumber).
				Msg("[SimAccessToken] failed to get token")
			return err
		}
		result.IsCreated = false
		return nil
	})

	return result, err
}
