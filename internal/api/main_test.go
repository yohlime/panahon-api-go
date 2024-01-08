package api

import (
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store, tokenMaker token.Maker) *Server {
	config := util.Config{
		GinMode:             gin.TestMode,
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		APIBasePath:         "/api/v1",
		EnableFileLogging:   false,
	}

	logger := util.NewLogger(config)

	server, err := NewServer(config, store, tokenMaker, logger)
	require.NoError(t, err)

	return server
}
