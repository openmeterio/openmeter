-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "expires_at" timestamptz NULL;
-- create "ledger_breakage_records" table
CREATE TABLE "ledger_breakage_records" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "kind" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "customer_id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "credit_priority" bigint NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "source_kind" character varying NOT NULL,
  "source_transaction_group_id" character(26) NULL,
  "source_transaction_id" character(26) NULL,
  "source_entry_id" character(26) NULL,
  "breakage_transaction_group_id" character(26) NOT NULL,
  "breakage_transaction_id" character(26) NOT NULL,
  "fbo_sub_account_id" character(26) NOT NULL,
  "breakage_sub_account_id" character(26) NOT NULL,
  "plan_id" character(26) NULL,
  "release_id" character(26) NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgerbreakagerecord_annotations" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_annotations" ON "ledger_breakage_records" USING gin ("annotations");
-- create index "ledgerbreakagerecord_id" to table: "ledger_breakage_records"
CREATE UNIQUE INDEX "ledgerbreakagerecord_id" ON "ledger_breakage_records" ("id");
-- create index "ledgerbreakagerecord_namespace" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace" ON "ledger_breakage_records" ("namespace");
-- create index "ledgerbreakagerecord_namespace_breakage_transaction_group_id" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace_breakage_transaction_group_id" ON "ledger_breakage_records" ("namespace", "breakage_transaction_group_id");
-- create index "ledgerbreakagerecord_namespace_customer_id_currency_credit_" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace_customer_id_currency_credit_" ON "ledger_breakage_records" ("namespace", "customer_id", "currency", "credit_priority", "expires_at", "id");
-- create index "ledgerbreakagerecord_namespace_plan_id" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace_plan_id" ON "ledger_breakage_records" ("namespace", "plan_id");
-- create index "ledgerbreakagerecord_namespace_source_entry_id" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace_source_entry_id" ON "ledger_breakage_records" ("namespace", "source_entry_id");
-- create index "ledgerbreakagerecord_namespace_source_transaction_group_id" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace_source_transaction_group_id" ON "ledger_breakage_records" ("namespace", "source_transaction_group_id");
