package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/middlewares"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateUserAccessRequest(t *testing.T) {
	user, _, _ := randomUser(t)
	tokenStr := gofakeit.LetterN(32)
	authUser := token.User{Username: user.Username}

	created := db.AccessRequest{
		ID:         7,
		UserID:     user.ID,
		StationIds: []byte("[1]"),
		Variables:  []byte(`["temp"]`),
		Status:     models.AccessRequestPending,
		Reason:     util.ToPgText("need access"),
		CreatedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	pending := db.AccessRequest{
		ID:        3,
		UserID:    user.ID,
		Status:    models.AccessRequestPending,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	deniedRecently := db.AccessRequest{
		ID:         4,
		UserID:     user.ID,
		Status:     models.AccessRequestDenied,
		CreatedAt:  pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		ReviewedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	deniedLongAgo := db.AccessRequest{
		ID:         5,
		UserID:     user.ID,
		Status:     models.AccessRequestDenied,
		CreatedAt:  pgtype.Timestamptz{Time: time.Now().Add(-26 * time.Hour), Valid: true},
		ReviewedAt: pgtype.Timestamptz{Time: time.Now().Add(-25 * time.Hour), Valid: true},
	}

	testCases := []struct {
		name         string
		existing     []db.AccessRequest
		expectCreate bool
		expectedCode int
	}{
		{name: "NoExistingRequest", existing: []db.AccessRequest{}, expectCreate: true, expectedCode: http.StatusCreated},
		{name: "PendingRequestBlocks", existing: []db.AccessRequest{pending}, expectedCode: http.StatusConflict},
		{name: "RecentDenialBlocks", existing: []db.AccessRequest{deniedRecently}, expectedCode: http.StatusTooManyRequests},
		{name: "OldDenialAllows", existing: []db.AccessRequest{deniedLongAgo}, expectCreate: true, expectedCode: http.StatusCreated},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tokenMaker := mocktoken.NewMockMaker(t)

			tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
			store.EXPECT().ListAccessRequestsByUser(mock.AnythingOfType("*gin.Context"), user.ID).
				Return(tc.existing, nil)
			if tc.expectCreate {
				store.EXPECT().CreateAccessRequest(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAccessRequestParams) bool {
					return arg.UserID == user.ID
				})).Return(created, nil)
			}

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST("/:id/access-requests",
				middlewares.AuthMiddleware(tokenMaker, store, false),
				handler.CreateUserAccessRequest)

			body := gin.H{
				"station_ids": []int64{1},
				"variables":   []string{"temp"},
				"reason":      "need access",
			}
			data, err := json.Marshal(body)
			require.NoError(t, err)

			url := fmt.Sprintf("/%d/access-requests", user.ID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)
			addAuthorization(request, AuthTypeBearer, tokenStr)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			store.AssertExpectations(t)
			require.Equal(t, tc.expectedCode, recorder.Code)
		})
	}
}
