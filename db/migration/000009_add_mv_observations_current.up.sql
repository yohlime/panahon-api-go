CREATE MATERIALIZED VIEW IF NOT EXISTS mv_observations_current
AS
    SELECT 
        DISTINCT(station_id),
        last_value(rr/6) OVER wdw::real AS rain,
        last_value("temp") OVER wdw::real AS "temp",
        last_value(rh) OVER wdw::real AS rh,
        last_value(wdir) OVER wdw::real AS wdir,
        last_value(wspd) OVER wdw::real AS wspd,
        last_value(srad) OVER wdw::real AS srad,
        last_value(mslp) OVER wdw::real AS mslp,
        first_value("temp") OVER t_wdw::real AS tn,
        last_value("temp") OVER t_wdw::real AS tx,
        last_value("wspdx") OVER w_wdw::real AS gust,
        sum(rr/6) OVER t_wdw::real AS rain_accum,
        first_value("timestamp") OVER t_wdw::timestamptz AS tn_timestamp,
        last_value("timestamp") OVER t_wdw::timestamptz AS tx_timestamp,
        last_value("timestamp") OVER w_wdw::timestamptz AS gust_timestamp,
        last_value("timestamp") OVER wdw::timestamptz AS "timestamp"
    FROM 
        observations_observation
    WHERE "timestamp" BETWEEN CURRENT_DATE AND CURRENT_TIMESTAMP
    WINDOW 
        wdw AS (PARTITION BY station_id ORDER BY "timestamp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
        t_wdw AS (PARTITION BY station_id ORDER BY "temp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING),
        w_wdw AS (PARTITION BY station_id ORDER BY "wspdx" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
WITH NO DATA;

CREATE UNIQUE INDEX "mv_observations_current_station_id_idx" ON mv_observations_current (station_id);

REFRESH MATERIALIZED VIEW mv_observations_current;
