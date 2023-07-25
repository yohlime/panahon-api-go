package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transaction
type Store interface {
	Querier
	FirstOrCreateSimAccessTokenTx(ctx context.Context, arg FirstOrCreateSimAccessTokenTxParams) (FirstOrCreateSimAccessTokenTxResult, error)
	BulkCreateUserRoles(ctx context.Context, arg []UserRolesParams) (ret []UserRolesParams, errs []error)
	BulkDeleteUserRoles(ctx context.Context, arg []UserRolesParams) []error
}

// SQLStore provides all functions to execute SQL queries and transactions
type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

// NewStore creates a new store
func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}
