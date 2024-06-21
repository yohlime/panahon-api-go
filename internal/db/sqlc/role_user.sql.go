// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: role_user.sql

package db

import (
	"context"
)

const listUserRoles = `-- name: ListUserRoles :one
SELECT ARRAY_AGG(r.name)::text[] role_names FROM role_user ru
JOIN roles r ON r.id = ru.role_id
WHERE user_id = $1
GROUP BY user_id
`

func (q *Queries) ListUserRoles(ctx context.Context, userID int64) ([]string, error) {
	row := q.db.QueryRow(ctx, listUserRoles, userID)
	var role_names []string
	err := row.Scan(&role_names)
	return role_names, err
}