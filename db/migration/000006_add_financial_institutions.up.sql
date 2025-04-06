CREATE TYPE financial_institution_type AS ENUM ('bank', 'wallet', 'mfi');

CREATE TABLE "financial_institutions" (
  "id" bigserial PRIMARY KEY,
  "name" varchar NOT NULL,
  "type" financial_institution_type NOT NULL,
  "logo_url" varchar NOT NULL,
  "code" varchar UNIQUE NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "financial_institutions" ("type");