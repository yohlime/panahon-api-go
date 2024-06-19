CREATE TABLE "sim_access_tokens" (
  "access_token" VARCHAR(255) PRIMARY KEY NOT NULL,
  "type" VARCHAR(50) NOT NULL,
  "mobile_number" VARCHAR(50) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

ALTER TABLE "sim_access_tokens"
  ADD CONSTRAINT "sim_access_tokens_mobile_number_fkey" FOREIGN KEY ("mobile_number") REFERENCES "sim_cards" ("mobile_number") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "sim_access_tokens_mobile_number_type_unique" UNIQUE ("mobile_number", "type");
