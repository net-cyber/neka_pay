ALTER TABLE "users" ADD COLUMN "token" varchar NOT NULL;
ALTER TABLE "users" ADD COLUMN "avatar" varchar NOT NULL;
ALTER TABLE "users" ADD COLUMN "fcmtoken" varchar NOT NULL;
ALTER TABLE "users" ADD COLUMN "online" boolean NOT NULL DEFAULT false;
