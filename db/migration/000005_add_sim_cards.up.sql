CREATE TABLE "sim_cards" (
  "mobile_number" VARCHAR(50) PRIMARY KEY NOT NULL,
  "type" VARCHAR(50),
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);
