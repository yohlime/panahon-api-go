CREATE TABLE "user_access" (
  "user_id" BIGINT PRIMARY KEY NOT NULL,
  "station_ids" JSONB,
  "variables" JSONB,
  "max_days_history" INTEGER NOT NULL DEFAULT 365,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE "access_requests" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "user_id" BIGINT NOT NULL,
  "station_ids" JSONB,
  "all_stations" BOOLEAN NOT NULL DEFAULT false,
  "variables" JSONB,
  "all_variables" BOOLEAN NOT NULL DEFAULT false,
  "max_days_history" INTEGER,
  "reason" TEXT,
  "status" VARCHAR(16) NOT NULL DEFAULT 'pending',
  "reviewed_by" BIGINT,
  "review_note" TEXT,
  "reviewed_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE INDEX "access_requests_user_id_idx" ON access_requests (user_id);
CREATE INDEX "access_requests_status_idx" ON access_requests (status);

ALTER TABLE "user_access"
  ADD CONSTRAINT "user_access_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "access_requests"
  ADD CONSTRAINT "access_requests_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "access_requests_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "users" ("id") ON DELETE SET NULL ON UPDATE CASCADE;
