-- name: ListLufftStationMsg :many
SELECT h.timestamp, h.message FROM observations_stationhealth h
WHERE h.station_id = $1
ORDER BY h.id
LIMIT $2
OFFSET $3;

-- name: CountLufftStationMsg :one
SELECT count(*) FROM observations_stationhealth h
WHERE h.station_id = $1;
