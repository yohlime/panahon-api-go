package handlers

import (
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type DefaultHandler struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	logger     *zerolog.Logger
}

func NewDefaultHandler(config util.Config, store db.Store, tokenMaker token.Maker, logger *zerolog.Logger) *DefaultHandler {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("mobile_number", validMobileNumber)
		v.RegisterValidation("fullname", validFullName)
		v.RegisterValidation("sentence", validSentence)
		v.RegisterValidation("date_time", validDateTimeStr)
	}

	return &DefaultHandler{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		logger:     logger,
	}
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

// stationOnlineThreshold is the configured freshness window for ONLINE status.
func (h *DefaultHandler) stationOnlineThreshold() time.Duration {
	if d := h.config.StationStatus.OnlineThreshold; d > 0 {
		return d
	}
	return time.Hour
}
