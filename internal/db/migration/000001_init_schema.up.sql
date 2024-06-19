CREATE TABLE "users" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "username" VARCHAR(255) NOT NULL,
  "full_name" VARCHAR(255) NOT NULL,
  "email" VARCHAR(255) NOT NULL,
  "password" VARCHAR(255) NOT NULL,
  "email_verified_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

ALTER TABLE "users"
  ADD CONSTRAINT "users_username_unique" UNIQUE ("username"),
  ADD CONSTRAINT "users_email_unique" UNIQUE ("email");
