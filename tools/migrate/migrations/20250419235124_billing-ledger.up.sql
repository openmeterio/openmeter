-- create "billing_ledgers" table
CREATE TABLE "billing_ledgers" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "currency" character varying NOT NULL,
  "customer_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_ledgers_customers_billing_ledger" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "billingledger_id" to table: "billing_ledgers"
CREATE UNIQUE INDEX "billingledger_id" ON "billing_ledgers" ("id");
-- create index "billingledger_namespace" to table: "billing_ledgers"
CREATE INDEX "billingledger_namespace" ON "billing_ledgers" ("namespace");
-- create index "billingledger_namespace_customer_id_currency" to table: "billing_ledgers"
CREATE UNIQUE INDEX "billingledger_namespace_customer_id_currency" ON "billing_ledgers" ("namespace", "customer_id", "currency") WHERE (deleted_at IS NULL);
-- create "billing_subledgers" table
CREATE TABLE "billing_subledgers" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "priority" bigint NOT NULL DEFAULT 0,
  "ledger_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_subledgers_billing_ledgers_subledgers" FOREIGN KEY ("ledger_id") REFERENCES "billing_ledgers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "billingsubledger_id" to table: "billing_subledgers"
CREATE UNIQUE INDEX "billingsubledger_id" ON "billing_subledgers" ("id");
-- create index "billingsubledger_namespace" to table: "billing_subledgers"
CREATE INDEX "billingsubledger_namespace" ON "billing_subledgers" ("namespace");
-- create index "billingsubledger_namespace_id" to table: "billing_subledgers"
CREATE UNIQUE INDEX "billingsubledger_namespace_id" ON "billing_subledgers" ("namespace", "id");
-- create index "billingsubledger_namespace_ledger_id_key" to table: "billing_subledgers"
CREATE UNIQUE INDEX "billingsubledger_namespace_ledger_id_key" ON "billing_subledgers" ("namespace", "ledger_id", "key") WHERE (deleted_at IS NULL);
-- create "billing_subledger_transactions" table
CREATE TABLE "billing_subledger_transactions" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "owner_type" character varying NULL,
  "owner_id" character varying NULL,
  "ledger_id" character(26) NOT NULL,
  "subledger_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_subledger_transactions_billing_ledgers_transactions" FOREIGN KEY ("ledger_id") REFERENCES "billing_ledgers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "billing_subledger_transactions_billing_subledgers_transactions" FOREIGN KEY ("subledger_id") REFERENCES "billing_subledgers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "billingsubledgertransaction_id" to table: "billing_subledger_transactions"
CREATE UNIQUE INDEX "billingsubledgertransaction_id" ON "billing_subledger_transactions" ("id");
-- create index "billingsubledgertransaction_ledger_id" to table: "billing_subledger_transactions"
CREATE INDEX "billingsubledgertransaction_ledger_id" ON "billing_subledger_transactions" ("ledger_id");
-- create index "billingsubledgertransaction_namespace" to table: "billing_subledger_transactions"
CREATE INDEX "billingsubledgertransaction_namespace" ON "billing_subledger_transactions" ("namespace");
-- create index "billingsubledgertransaction_namespace_id" to table: "billing_subledger_transactions"
CREATE UNIQUE INDEX "billingsubledgertransaction_namespace_id" ON "billing_subledger_transactions" ("namespace", "id");
-- create index "billingsubledgertransaction_subledger_id" to table: "billing_subledger_transactions"
CREATE INDEX "billingsubledgertransaction_subledger_id" ON "billing_subledger_transactions" ("subledger_id");
