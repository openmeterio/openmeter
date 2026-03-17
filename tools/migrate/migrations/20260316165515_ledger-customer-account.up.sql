-- create "ledger_customer_accounts" table
CREATE TABLE "ledger_customer_accounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NOT NULL,
  "account_type" character varying NOT NULL,
  "account_id" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgercustomeraccount_id" to table: "ledger_customer_accounts"
CREATE UNIQUE INDEX "ledgercustomeraccount_id" ON "ledger_customer_accounts" ("id");
-- create index "ledgercustomeraccount_namespace" to table: "ledger_customer_accounts"
CREATE INDEX "ledgercustomeraccount_namespace" ON "ledger_customer_accounts" ("namespace");
-- create index "ledgercustomeraccount_namespace_customer_id_account_type" to table: "ledger_customer_accounts"
CREATE UNIQUE INDEX "ledgercustomeraccount_namespace_customer_id_account_type" ON "ledger_customer_accounts" ("namespace", "customer_id", "account_type");
-- create index "ledgercustomeraccount_namespace_id" to table: "ledger_customer_accounts"
CREATE UNIQUE INDEX "ledgercustomeraccount_namespace_id" ON "ledger_customer_accounts" ("namespace", "id");
