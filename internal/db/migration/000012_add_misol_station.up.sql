CREATE TABLE "misol_station" (
  "id" BIGINT PRIMARY KEY NOT NULL,
  "station_id" BIGINT NOT NULL
);

ALTER TABLE "misol_station"
  ADD CONSTRAINT "misol_station_station_id_fkey" FOREIGN KEY ("station_id") REFERENCES "observations_station" ("id") ON DELETE CASCADE ON UPDATE CASCADE;
