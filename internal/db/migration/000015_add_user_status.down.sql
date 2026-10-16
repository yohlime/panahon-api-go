ALTER TABLE "users" DROP CONSTRAINT "users_reviewed_by_fkey";
DROP INDEX "users_status_idx";
ALTER TABLE "users"
  DROP COLUMN "status",
  DROP COLUMN "reviewed_by",
  DROP COLUMN "reviewed_at",
  DROP COLUMN "review_note";
