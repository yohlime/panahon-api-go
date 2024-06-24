package handlers

import (
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
)

func newTestHandler(store db.Store, tokenMaker token.Maker) *DefaultHandler {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		EnableFileLogging:   false,
	}

	logger := util.NewLogger(config)

	gin.SetMode(gin.TestMode)

	return NewDefaultHandler(config, store, tokenMaker, logger)
}
