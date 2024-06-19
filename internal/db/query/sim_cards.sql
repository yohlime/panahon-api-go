-- name: CreateSimCard :one
INSERT INTO sim_cards (
  mobile_number,
  type
) VALUES (
  $1, $2
) RETURNING *;

-- name: GetSimCard :one
SELECT * FROM sim_cards
WHERE mobile_number = $1 LIMIT 1;