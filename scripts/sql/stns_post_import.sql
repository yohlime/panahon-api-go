BEGIN;

UPDATE observations_station
SET created_at = '0001-01-01 00:00:00+00'
WHERE created_at IS NULL;

UPDATE observations_station
SET deleted_at = '0001-01-01 00:00:00+00'
WHERE deleted_at IS NULL;

ALTER TABLE observations_station
ALTER COLUMN created_at SET NOT NULL,
ALTER COLUMN deleted_at SET NOT NULL;

COMMIT;
