package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

const (
	GLabsTokenFetchURL    = "https://developer.globelabs.com.ph/oauth/access_token"
	GLabsAccessTokenType  = "GLABS"
	GLabsMobileNumberType = "GLOBE"
)

type gLabsOptInReq struct {
	AccessToken      string `form:"access_token" json:"access_token" binding:"required_with=SubscriberNumber"`
	SubscriberNumber string `form:"subscriber_number" json:"subscriber_number" binding:"required_with=AccessToken,omitempty,number,len=10"`
	Code             string `form:"code" json:"code"`
}

// GLabsOptIn godoc
// @Summary      Globe Labs opt-in
// @Tags         GLabs
// @Produce      json
// @Success      200 {object} simAccessToken
// @Router       /glabs/optin [get]
func (s *Server) GLabsOptIn(ctx *gin.Context) {
	var req gLabsOptInReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		log.Error().Err(err).
			Msg("[GLabs] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Opt-in via web form
	if req.Code != "" {
		log.Debug().Msg("[GLabs] Opt-in via web form")
		err := req.fetchGLabsAccessToken(s.config.GlabsAppID, s.config.GlabsAppSecret)
		if err != nil {
			log.Error().Err(err).
				Msg("[GLabs] Cannot retrieve access token")
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	// Opt-in via SMS
	if req.SubscriberNumber != "" && req.AccessToken != "" {
		log.Debug().Msg("[GLabs] Opt-in via SMS")

		arg := db.FirstOrCreateSimAccessTokenTxParams{
			AccessToken:     req.AccessToken,
			AccessTokenType: GLabsAccessTokenType,
			MobileNumber:    fmt.Sprintf("63%s", req.SubscriberNumber),
			MobileNumberType: util.NullString{
				Text: pgtype.Text{
					String: GLabsMobileNumberType,
					Valid:  true,
				},
			},
		}

		simAccessToken, err := s.store.FirstOrCreateSimAccessTokenTx(ctx, arg)
		if err != nil {
			log.Error().Err(err).
				Str("mobile_number", req.SubscriberNumber).
				Msg("[GLabs] Cannot retrieve or create access token")
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		log.Debug().
			Str("mobile_number", req.SubscriberNumber).
			Msg("[GLabs] Mobile number registered successfully")
		ctx.JSON(http.StatusCreated, simAccessToken)
		return
	}

	ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("missing parameters")))
}

type gLabsLoadReq struct {
	OutboundRewardRequest struct {
		TransactionID int    `json:"transaction_id"`
		Status        string `json:"status"`
		Address       string `json:"address"`
		Promo         string `json:"promo"`
		Timestamp     string `json:"timestamp"`
	} `json:"outboundRewardRequest"`
}

// CreateGLabsLoad godoc
// @Summary      Create Globe Labs entry
// @Tags         GLabs
// @Produce      json
// @Success      200 {object} db.GLabsLoad
// @Router       /glabs/load [post]
func (s *Server) CreateGLabsLoad(ctx *gin.Context) {
	var req gLabsLoadReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).
			Msg("[GLabsLoad] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Debug().
		Str("subscriber", req.OutboundRewardRequest.Address).
		Str("promo", req.OutboundRewardRequest.Promo).
		Str("status", req.OutboundRewardRequest.Status).
		Msg("[GLabsLoad] Top up detected")

	mobileNumber := "63" + req.OutboundRewardRequest.Address
	_, err := s.store.GetStationByMobileNumber(ctx, util.NullString{
		Text: pgtype.Text{
			String: mobileNumber,
			Valid:  true,
		},
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			log.Warn().Err(err).
				Str("subscriber", mobileNumber).
				Str("promo", req.OutboundRewardRequest.Promo).
				Str("status", req.OutboundRewardRequest.Status).
				Msg("[GLabsLoad] Unknown number")
		}
	}

	gLabsLoad, err := s.store.CreateGLabsLoad(ctx, db.CreateGLabsLoadParams{
		Promo: util.NullString{
			Text: pgtype.Text{
				String: req.OutboundRewardRequest.Promo,
				Valid:  true,
			},
		},
		TransactionID: util.NullInt4{
			Int4: pgtype.Int4{
				Int32: int32(req.OutboundRewardRequest.TransactionID),
				Valid: true,
			},
		},
		Status: util.NullString{
			Text: pgtype.Text{
				String: req.OutboundRewardRequest.Status,
				Valid:  true,
			},
		},
		MobileNumber: mobileNumber,
	})
	if err != nil {
		log.Error().Err(err).
			Str("sender", mobileNumber).
			Str("promo", req.OutboundRewardRequest.Promo).
			Str("status", req.OutboundRewardRequest.Status).
			Msg("[GLabsLoad] Error occured")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, gLabsLoad)
}

type fetchGLabsAccessTokenReq struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Code      string `json:"code"`
}

func (g *gLabsOptInReq) fetchGLabsAccessToken(appID, appSecret string) error {
	req := fetchGLabsAccessTokenReq{
		AppID:     appID,
		AppSecret: appSecret,
		Code:      g.Code,
	}
	json_data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		GLabsTokenFetchURL,
		"application/json",
		bytes.NewBuffer(json_data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var gotGTok gLabsOptInReq
	if err := json.NewDecoder(resp.Body).Decode(&gotGTok); err != nil {
		return err
	}

	g.AccessToken = gotGTok.AccessToken
	g.SubscriberNumber = gotGTok.SubscriberNumber
	return nil
}
