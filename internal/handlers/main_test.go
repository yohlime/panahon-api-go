package handlers

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
)

const (
	AuthTypeBearer = "bearer"
	AuthTypeCookie = "cookie"
	AuthTypeQuery  = "query"
)

func newTestHandler(store db.Store, tokenMaker token.Maker) *DefaultHandler {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		EnableFileLogging:   false,
		StationStatus:       util.StationStatusConfig{OnlineThreshold: time.Hour},
	}

	logger := util.NewLogger(config)

	gin.SetMode(gin.TestMode)

	return NewDefaultHandler(config, store, tokenMaker, logger)
}

func addAuthorization(
	request *http.Request,
	authType string,
	token string,
) {
	if authType == AuthTypeBearer {
		authorizationHeader := fmt.Sprintf("%s %s", authType, token)
		request.Header.Set(models.AuthHeaderKey, authorizationHeader)
	} else if authType == AuthTypeCookie {
		addAccessTokenCookie(request, token)
	} else if authType == AuthTypeQuery {
		request.URL.RawQuery = fmt.Sprintf("%s=%s", models.APIKeyQueryParam, token)
	}
}
