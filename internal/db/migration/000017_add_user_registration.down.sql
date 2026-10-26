DROP TABLE "user_tokens";

DROP INDEX "users_email_lower_unique";
ALTER TABLE "users" ADD CONSTRAINT "users_email_unique" UNIQUE ("email");
ALTER TABLE "users" ALTER COLUMN "email" SET NOT NULL;
