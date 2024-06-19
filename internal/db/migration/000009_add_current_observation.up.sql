CREATE TABLE "observations_current" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "station_id" BIGINT NOT NULL NOT NULL,
  "rain" REAL,
  "temp" REAL,
  "rh" REAL,
  "wdir" REAL,
  "wspd" REAL,
  "srad" REAL,
  "mslp" REAL,
  "tn" REAL,
  "tx" REAL,
  "gust" REAL,
  "rain_accum" REAL,
  "timestamp" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "tn_timestamp" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "tx_timestamp" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "gust_timestamp" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

ALTER TABLE "observations_current"
  ADD CONSTRAINT "observations_current_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "observations_current_station_id_timestamp_unique" UNIQUE ("station_id", "timestamp");