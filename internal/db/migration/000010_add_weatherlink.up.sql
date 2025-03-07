CREATE TABLE "weatherlink" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "station_id" BIGINT NOT NULL NOT NULL,
  "uuid" VARCHAR(255),
  "api_key" VARCHAR(255),
  "api_secret" VARCHAR(255),
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "deleted_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

ALTER TABLE "weatherlink"
  ADD CONSTRAINT "weatherlink_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE;
