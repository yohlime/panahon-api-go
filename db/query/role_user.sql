-- name: BatchCreateUserRoles :batchone
INSERT INTO role_user (user_id, role_id)
SELECT u.id, r.id
FROM (
	VALUES
		(@username::text, @role_name::text)
) ru (username, role_name)
JOIN users u USING(username)
JOIN roles r ON ru.role_name = r.name
RETURNING *;

-- name: ListUserRoles :one
SELECT ARRAY_AGG(r.name)::text[] role_names FROM role_user ru
JOIN roles r ON r.id = ru.role_id
WHERE user_id = $1
GROUP BY user_id;

-- name: BatchDeleteUserRoles :batchexec
DELETE FROM role_user AS ru
USING
	users AS u,
	roles AS r
WHERE 
	u.id = ru.user_id
	AND r.id = ru.role_id
	AND u.username = @username::text
	AND r.name = @role_name::text;
