-- name: CreateUser :one
INSERT INTO users (
  username,
  password,
  full_name,
  email,
  status
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'));

-- name: UpdateUser :one
UPDATE users
SET
  username = COALESCE(sqlc.narg(username), username),
  password = COALESCE(sqlc.narg(password), password),
  password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
  full_name = COALESCE(sqlc.narg(full_name), full_name),
  email = COALESCE(sqlc.narg(email), email),
  updated_at = now()
WHERE
  id = sqlc.arg(id)
RETURNING *;

-- name: ReviewUser :one
UPDATE users
SET
  status = sqlc.arg(status),
  reviewed_by = sqlc.narg(reviewed_by),
  review_note = sqlc.narg(review_note),
  reviewed_at = now(),
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
