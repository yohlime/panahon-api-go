CREATE TABLE "roles" (
  "id" BIGSERIAL PRIMARY KEY NOT NULL,
  "name" VARCHAR(255) NOT NULL,
  "description" VARCHAR(255),
  "created_at" timestamptz DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE "role_user" (
  "role_id" BIGINT NOT NULL,
  "user_id" BIGINT NOT NULL,
  "created_at" timestamptz DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz DEFAULT '0001-01-01 00:00:00Z',
  PRIMARY KEY ("user_id", "role_id")
);

ALTER TABLE "roles"
  ADD CONSTRAINT "roles_name_unique" UNIQUE ("name");

ALTER TABLE "role_user"
  ADD CONSTRAINT "role_user_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT "role_user_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE CASCADE ON UPDATE CASCADE;
