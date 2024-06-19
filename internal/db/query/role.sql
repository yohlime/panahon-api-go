-- name: CreateRole :one
INSERT INTO roles (
  name,
  description
) VALUES (
  $1, $2
) RETURNING *;

-- name: GetRole :one
SELECT * FROM roles
WHERE id = $1 LIMIT 1;

-- name: GetRoleByName :one
SELECT * FROM roles
WHERE name = $1 LIMIT 1;

-- name: ListRoles :many
SELECT * FROM roles
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: CountRoles :one
SELECT count(*) FROM roles;

-- name: UpdateRole :one
UPDATE roles
SET
  name = COALESCE(sqlc.narg(name), name),
  description = COALESCE(sqlc.narg(description), description),
  updated_at = now()
WHERE
  id = sqlc.arg(id)
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;