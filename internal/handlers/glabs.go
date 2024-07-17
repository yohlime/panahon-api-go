package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	GLabsTokenFetchURL    = "https://developer.globelabs.com.ph/oauth/access_token"
	GLabsAccessTokenType  = "GLABS"
	GLabsMobileNumberType = "GLOBE"
)

type gLabsOptInReq struct {
	AccessToken      string `form:"access_token" json:"access_token" binding:"required_with=SubscriberNumber" fake:"{lettern:32}"`
	SubscriberNumber string `form:"subscriber_number" json:"subscriber_number" binding:"required_with=AccessToken,omitempty" fake:"{regex:9[0-9]{9}}"`
	Code             string `form:"code" json:"code" fake:"{lettern:8}"`
} //@name GlobeLabsOptInParams

type gLabsOptInRes struct {
	AccessToken  string `json:"access_token"`
	Type         string `json:"type"`
	MobileNumber string `json:"mobile_number"`
	IsCreated    bool   `json:"is_created"`
} //@name GlobeLabsOptInResponse

func newGLabsOptInResponse(res db.FirstOrCreateSimAccessTokenTxResult) gLabsOptInRes {
	return gLabsOptInRes{
		AccessToken:  res.AccessToken.AccessToken,
		Type:         res.AccessToken.Type,
		MobileNumber: res.AccessToken.MobileNumber,
		IsCreated:    res.IsCreated,
	}
}

// GLabsOptIn
//
//	@Summary	Globe Labs opt-in
//	@Tags		globelabs
//	@Accept		json
//	@Produce	json
//	@Param		req	query		gLabsOptInReq	true	"Globe Labs Opt-in query"
//	@Success	200	{object}	gLabsOptInRes
//	@Router		/glabs [get]
func (h *DefaultHandler) GLabsOptIn(ctx *gin.Context) {
	var req gLabsOptInReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).
			Msg("[GLabs] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Opt-in via web form
	if req.Code != "" {
		h.logger.Debug().Msg("[GLabs] Opt-in via web form")
		err := req.fetchGLabsAccessToken(h.config.GlabsAppID, h.config.GlabsAppSecret)
		if err != nil {
			h.logger.Error().Err(err).
				Msg("[GLabs] Cannot retrieve access token")
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	// Opt-in via SMS
	if req.SubscriberNumber != "" && req.AccessToken != "" {
		h.logger.Debug().Msg("[GLabs] Opt-in via SMS")

		mobileNumber, ok := util.ParseMobileNumber(req.SubscriberNumber)
		if !ok {
			err := fmt.Errorf("invalid mobile number: %s", req.SubscriberNumber)
			h.logger.Error().Err(err).
				Str("mobile_number", req.SubscriberNumber).
				Msg("[GLabs] Cannot create access token")
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		arg := db.FirstOrCreateSimAccessTokenTxParams{
			AccessToken:     req.AccessToken,
			AccessTokenType: GLabsAccessTokenType,
			MobileNumber:    mobileNumber,
			MobileNumberType: pgtype.Text{
				String: GLabsMobileNumberType,
				Valid:  true,
			},
		}

		simAccessToken, err := h.store.FirstOrCreateSimAccessTokenTx(ctx, arg)
		if err != nil {
			h.logger.Error().Err(err).
				Str("mobile_number", req.SubscriberNumber).
				Msg("[GLabs] Cannot retrieve or create access token")
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		h.logger.Debug().
			Str("mobile_number", req.SubscriberNumber).
			Msg("[GLabs] Mobile number registered successfully")
		ctx.JSON(http.StatusCreated, newGLabsOptInResponse(simAccessToken))
		return
	}

	ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("missing parameters")))
}

type gLabsUnsubscribeReq struct {
	Unsubscribed struct {
		SubscriberNumber string `json:"subscriber_number" binding:"required"`
		AccessToken      string `json:"access_token" binding:"required"`
		Timestamp        string `json:"time_stamp"`
	} `json:"unsubscribed"`
} //@name GlobeLabsUnsubscribeParams

// GLabsUnsubscribe
//
//	@Summary	Globe Labs unsubscribe
//	@Tags		globelabs
//	@Accept		json
//	@Produce	json
//	@Param		req	query	gLabsUnsubscribeReq	true	"Globe Labs unsubscribe params"
//	@Success	204
//	@Router		/glabs [post]
func (h *DefaultHandler) GLabsUnsubscribe(ctx *gin.Context) {
	var req gLabsUnsubscribeReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).
			Msg("[GLabs] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	mobileNumber, ok := util.ParseMobileNumber(req.Unsubscribed.SubscriberNumber)
	if !ok {
		err := fmt.Errorf("invalid mobile number: %s", req.Unsubscribed.SubscriberNumber)
		h.logger.Error().Err(err).
			Str("mobile_number", req.Unsubscribed.SubscriberNumber).
			Str("access_token_type", GLabsAccessTokenType).
			Msg("[GLabs] Cannot remove access token")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := h.store.DeleteSimAccessToken(ctx, req.Unsubscribed.AccessToken)
	if err != nil {
		h.logger.Error().Err(err).
			Str("mobile_number", mobileNumber).
			Str("access_token_type", GLabsAccessTokenType).
			Msg("[GLabs] Cannot remove access token")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	h.logger.Debug().
		Str("mobile_number", mobileNumber).
		Str("access_token_type", GLabsAccessTokenType).
		Msg("[GLabs] Mobile number unregistered successfully")
	ctx.JSON(http.StatusNoContent, nil)
}

type gLabsLoadReq struct {
	OutboundRewardRequest struct {
		TransactionID int    `json:"transaction_id"`
		Status        string `json:"status"`
		Address       string `json:"address"`
		Promo         string `json:"promo"`
		Timestamp     string `json:"timestamp"`
	} `json:"outboundRewardRequest"`
} //@name GlobeLabsLoadParams

type gLabsLoadRes struct {
	Status        string             `json:"status"`
	Promo         string             `json:"promo"`
	TransactionID int32              `json:"transaction_id"`
	MobileNumber  string             `json:"mobile_number"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
} //@name GlobeLabsLoadResponse

func newGLabsLoadResponse(g db.GlabsLoad) gLabsLoadRes {
	res := gLabsLoadRes{
		MobileNumber: g.MobileNumber,
		CreatedAt:    g.CreatedAt,
	}

	if g.TransactionID.Valid {
		res.TransactionID = g.TransactionID.Int32
	}
	if g.Status.Valid {
		res.Status = g.Status.String
	}
	if g.Promo.Valid {
		res.Promo = g.Promo.String
	}

	return res
}

// CreateGLabsLoad
//
//	@Summary	Create Globe Labs entry
//	@Tags		globelabs
//	@Accept		json
//	@Produce	json
//	@Param		req	query		gLabsLoadReq	true	"Globe Labs Load query"
//	@Success	200	{object}	gLabsLoadRes
//	@Router		/glabs/load [post]
func (h *DefaultHandler) CreateGLabsLoad(ctx *gin.Context) {
	var req gLabsLoadReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).
			Msg("[GLabsLoad] Bad request")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	h.logger.Debug().
		Str("subscriber", req.OutboundRewardRequest.Address).
		Str("promo", req.OutboundRewardRequest.Promo).
		Str("status", req.OutboundRewardRequest.Status).
		Msg("[GLabsLoad] Top up detected")

	mobileNumber, ok := util.ParseMobileNumber(req.OutboundRewardRequest.Address)
	if !ok {
		err := fmt.Errorf("invalid mobile number: %s", req.OutboundRewardRequest.Address)
		h.logger.Error().Err(err).
			Str("mobile_number", req.OutboundRewardRequest.Address).
			Msg("[GLabsLoad] Invalid number")
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, err := h.store.GetStationByMobileNumber(ctx,
		pgtype.Text{
			String: mobileNumber,
			Valid:  true,
		},
	)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			h.logger.Warn().Err(err).
				Str("subscriber", mobileNumber).
				Str("promo", req.OutboundRewardRequest.Promo).
				Str("status", req.OutboundRewardRequest.Status).
				Msg("[GLabsLoad] Unknown number")
		}
	}

	gLabsLoad, err := h.store.CreateGLabsLoad(ctx, db.CreateGLabsLoadParams{
		Promo: pgtype.Text{
			String: req.OutboundRewardRequest.Promo,
			Valid:  true,
		},
		TransactionID: pgtype.Int4{
			Int32: int32(req.OutboundRewardRequest.TransactionID),
			Valid: true,
		},
		Status: pgtype.Text{
			String: req.OutboundRewardRequest.Status,
			Valid:  true,
		},
		MobileNumber: mobileNumber,
	})
	if err != nil {
		h.logger.Error().Err(err).
			Str("sender", mobileNumber).
			Str("promo", req.OutboundRewardRequest.Promo).
			Str("status", req.OutboundRewardRequest.Status).
			Msg("[GLabsLoad] Error occured")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newGLabsLoadResponse(gLabsLoad))
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
