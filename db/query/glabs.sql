-- name: CreateGLabsLoad :one
INSERT INTO glabs_load (
  promo,
  transaction_id,
  status,
  mobile_number
) VALUES (
  $1, $2, $3, $4
) RETURNING *;
