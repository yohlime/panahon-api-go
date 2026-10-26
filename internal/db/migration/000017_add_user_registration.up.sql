-- Email is optional for admin-created accounts. Multiple users may have no
-- email, but any non-null email must still be unique (case-insensitively).
ALTER TABLE "users" ALTER COLUMN "email" DROP NOT NULL;
ALTER TABLE "users" DROP CONSTRAINT "users_email_unique";

-- Treat pre-existing empty strings as "no email" and normalize existing
-- addresses before building the partial unique index.
UPDATE "users" SET "email" = NULL WHERE "email" = '';
UPDATE "users" SET "email" = lower(trim("email")) WHERE "email" IS NOT NULL;
DO $$
BEGIN
  IF EXISTS (
    SELECT lower("email")
    FROM "users"
    WHERE "email" IS NOT NULL
    GROUP BY lower("email")
    HAVING count(*) > 1
  ) THEN
    RAISE EXCEPTION 'users.email has case-insensitive duplicates; resolve them before migrating';
  END IF;
END $$;

CREATE UNIQUE INDEX "users_email_lower_unique"
  ON "users" (lower("email"))
  WHERE "email" IS NOT NULL;

-- One-time tokens for admin-created users to finish registration (set their
-- password, confirm their email, and activate the account).
CREATE TABLE "user_tokens" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "user_id" BIGINT NOT NULL,
  "purpose" VARCHAR(32) NOT NULL,
  "token_hash" VARCHAR(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz,
  "created_by" BIGINT,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

ALTER TABLE "user_tokens"
  ADD CONSTRAINT "user_tokens_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "user_tokens_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT "user_tokens_token_hash_unique" UNIQUE ("token_hash");

CREATE INDEX "user_tokens_user_purpose_idx" ON "user_tokens" ("user_id", "purpose");
CREATE INDEX "user_tokens_expires_at_idx" ON "user_tokens" ("expires_at");
