ALTER TABLE "observations_station" ADD COLUMN geom GEOMETRY(Point, 4326);
UPDATE "observations_station" SET geom = ST_POINT(lon, lat, 4326);

CREATE INDEX "observations_station_geom_idx" ON "observations_station" USING gist (geom);
