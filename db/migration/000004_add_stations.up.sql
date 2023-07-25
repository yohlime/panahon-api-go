CREATE TABLE "observations_station" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "name" VARCHAR(200) NOT NULL,
  "lat" REAL,
  "lon" REAL,
  "elevation" REAL,
  "date_installed" DATE,
  "mo_station_id" VARCHAR(50),
  "sms_system_type" VARCHAR(50),
  "mobile_number" VARCHAR(50),
  "station_type" VARCHAR(50),
  "station_type2" VARCHAR(50),
  "station_url" VARCHAR(255),
  "status" VARCHAR(50),
  "logger_version" VARCHAR(255),
  "priority_level" VARCHAR(255),
  "provider_id" VARCHAR(255),
  "province" VARCHAR(255),
  "region" VARCHAR(255),
  "address" VARCHAR,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "deleted_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE "observations_mo_observation" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "pres" REAL,
  "rr" REAL,
  "rh" REAL,
  "temp" REAL,
  "td" REAL,
  "wdir" REAL,
  "wspd" REAL,
  "wspdx" REAL,
  "srad" REAL,
  "hi" REAL,
  "station_id" BIGINT NOT NULL,
  "timestamp" timestamptz,
  "wchill" REAL,
  "rain" REAL,
  "tx" REAL,
  "tn" REAL,
  "wrun" REAL,
  "thwi" REAL,
  "thswi" REAL,
  "senergy" REAL,
  "sradx" REAL,
  "uvi" REAL,
  "uvdose" REAL,
  "uvx" REAL,
  "hdd" REAL,
  "cdd" REAL,
  "et" REAL,
  "qc_level" INTEGER NOT NULL DEFAULT 0,
  "wdirx" REAL,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE "observations_observation" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "pres" REAL,
  "rr" REAL,
  "rh" REAL,
  "temp" REAL,
  "td" REAL,
  "wdir" REAL,
  "wspd" REAL,
  "wspdx" REAL,
  "srad" REAL,
  "mslp" REAL,
  "hi" REAL,
  "station_id" BIGINT NOT NULL,
  "timestamp" timestamptz,
  "wchill" REAL,
  "qc_level" INTEGER NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE "observations_stationhealth" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "vb1" REAL,
  "vb2" REAL,
  "curr" REAL,
  "bp1" REAL,
  "bp2" REAL,
  "cm" VARCHAR(255),
  "ss" INTEGER,
  "temp_arq" REAL,
  "rh_arq" REAL,
  "fpm" VARCHAR(255),
  "error_msg" TEXT,
  "message" TEXT,
  "data_count" INTEGER,
  "data_status" VARCHAR(255),
  "timestamp" timestamptz(6),
  "station_id" BIGINT NOT NULL,
  "minutes_difference" INTEGER,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

-- CREATE TABLE "observations_derivedhourly" (
--   "id" BIGSERIAL PRIMARY KEY NOT NULL,
--   "pres" REAL,
--   "rr" REAL,
--   "rh" REAL,
--   "temp" REAL,
--   "td" REAL,
--   "wspd" REAL,
--   "wdir" REAL,
--   "srad" REAL,
--   "hi" REAL,
--   "uvi" REAL,
--   "timestamp" timestamptz(0) NOT NULL,
--   "station_id" BIGINT NOT NULL NOT NULL,
--   "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
--   "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
-- );

ALTER TABLE "observations_station"
  ADD CONSTRAINT "observations_station_mobile_number_unique" UNIQUE ("mobile_number");

ALTER TABLE "observations_observation"
  ADD CONSTRAINT "observations_observation_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "observations_observation_station_id_timestamp_unique" UNIQUE ("station_id", "timestamp");

ALTER TABLE "observations_stationhealth"
  ADD CONSTRAINT "observations_stationhealth_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "observations_stationhealth_station_id_timestamp_unique" UNIQUE ("station_id", "timestamp");

ALTER TABLE "observations_mo_observation"
  ADD CONSTRAINT "observations_mo_observation_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "observations_mo_observation_station_id_timestamp_unique" UNIQUE ("station_id", "timestamp");

-- CREATE INDEX "observations_derivedhourly_idx_station_id_timestamp" ON "observations_derivedhourly" ("station_id", "timestamp");

-- CREATE UNIQUE INDEX "observations_derivedhourly_station_id_timestamp_unique" ON "observations_derivedhourly" ("station_id", "timestamp");

-- ALTER TABLE "observations_derivedhourly" ADD CONSTRAINT "observations_derivedhourly_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE;
