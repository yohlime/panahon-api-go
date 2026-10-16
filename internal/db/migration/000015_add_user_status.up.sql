ALTER TABLE "users"
  ADD COLUMN "status" VARCHAR(16) NOT NULL DEFAULT 'active',
  ADD COLUMN "reviewed_by" BIGINT,
  ADD COLUMN "reviewed_at" timestamptz,
  ADD COLUMN "review_note" TEXT;

CREATE INDEX "users_status_idx" ON users (status);

ALTER TABLE "users"
  ADD CONSTRAINT "users_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "users" ("id") ON DELETE SET NULL ON UPDATE CASCADE;
