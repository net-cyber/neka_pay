CREATE TABLE "external_transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_bank_code" varchar NOT NULL,
  "to_account_number" varchar NOT NULL,
  "recipient_name" varchar NOT NULL,
  "amount" bigint NOT NULL,
  "currency" varchar NOT NULL,
  "status" varchar NOT NULL DEFAULT 'pending',
  "reference" varchar NOT NULL,
  "description" varchar,
  "transaction_id" varchar,
  "transaction_fees" bigint DEFAULT 0,
  "error_message" varchar,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "external_transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");

CREATE INDEX ON "external_transfers" ("from_account_id");
CREATE INDEX ON "external_transfers" ("reference");
CREATE INDEX ON "external_transfers" ("transaction_id");
CREATE INDEX ON "external_transfers" ("status");

COMMENT ON COLUMN "external_transfers"."amount" IS 'must be positive';
COMMENT ON COLUMN "external_transfers"."status" IS 'pending, processing, completed, failed'; 