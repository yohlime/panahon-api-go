// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: role.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const countRoles = `-- name: CountRoles :one
SELECT count(*) FROM roles
`

func (q *Queries) CountRoles(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, countRoles)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createRole = `-- name: CreateRole :one
INSERT INTO roles (
  name,
  description
) VALUES (
  $1, $2
) RETURNING id, name, description, created_at, updated_at
`

type CreateRoleParams struct {
	Name        string      `json:"name"`
	Description pgtype.Text `json:"description"`
}

func (q *Queries) CreateRole(ctx context.Context, arg CreateRoleParams) (Role, error) {
	row := q.db.QueryRow(ctx, createRole, arg.Name, arg.Description)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteRole = `-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1
`

func (q *Queries) DeleteRole(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, deleteRole, id)
	return err
}

const getRole = `-- name: GetRole :one
SELECT id, name, description, created_at, updated_at FROM roles
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetRole(ctx context.Context, id int64) (Role, error) {
	row := q.db.QueryRow(ctx, getRole, id)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getRoleByName = `-- name: GetRoleByName :one
SELECT id, name, description, created_at, updated_at FROM roles
WHERE name = $1 LIMIT 1
`

func (q *Queries) GetRoleByName(ctx context.Context, name string) (Role, error) {
	row := q.db.QueryRow(ctx, getRoleByName, name)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listRoles = `-- name: ListRoles :many
SELECT id, name, description, created_at, updated_at FROM roles
ORDER BY id
LIMIT $1
OFFSET $2
`

type ListRolesParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func (q *Queries) ListRoles(ctx context.Context, arg ListRolesParams) ([]Role, error) {
	rows, err := q.db.Query(ctx, listRoles, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Role{}
	for rows.Next() {
		var i Role
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateRole = `-- name: UpdateRole :one
UPDATE roles
SET
  name = COALESCE($1, name),
  description = COALESCE($2, description),
  updated_at = now()
WHERE
  id = $3
RETURNING id, name, description, created_at, updated_at
`

type UpdateRoleParams struct {
	Name        pgtype.Text `json:"name"`
	Description pgtype.Text `json:"description"`
	ID          int64       `json:"id"`
}

func (q *Queries) UpdateRole(ctx context.Context, arg UpdateRoleParams) (Role, error) {
	row := q.db.QueryRow(ctx, updateRole, arg.Name, arg.Description, arg.ID)
	var i Role
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
