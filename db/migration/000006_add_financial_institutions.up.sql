DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'financial_institution_type') THEN
        CREATE TYPE financial_institution_type AS ENUM ('bank', 'wallet', 'mfi');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS "financial_institutions" (
  "id" bigserial PRIMARY KEY,
  "name" varchar NOT NULL,
  "type" financial_institution_type NOT NULL,
  "logo_url" varchar NOT NULL,
  "code" varchar UNIQUE NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX IF NOT EXISTS financial_institutions_type_idx ON "financial_institutions" ("type");