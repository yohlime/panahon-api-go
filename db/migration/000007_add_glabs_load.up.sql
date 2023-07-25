CREATE TABLE "glabs_load" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "status" VARCHAR(255),
  "promo" VARCHAR(255),
  "transaction_id" INTEGER,
  "mobile_number" VARCHAR(50) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);
