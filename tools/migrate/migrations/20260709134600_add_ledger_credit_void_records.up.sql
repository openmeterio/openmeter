-- create "ledger_credit_void_records" table
CREATE TABLE "ledger_credit_void_records" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "amount" numeric NOT NULL,
  "customer_id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "voided_at" timestamptz NOT NULL,
  "source_charge_id" character(26) NOT NULL,
  "void_transaction_group_id" character(26) NOT NULL,
  "void_transaction_id" character(26) NOT NULL,
  "fbo_sub_account_id" character(26) NOT NULL,
  "receivable_sub_account_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgercreditvoidrecord_annotations" to table: "ledger_credit_void_records"
CREATE INDEX "ledgercreditvoidrecord_annotations" ON "ledger_credit_void_records" USING gin ("annotations");
-- create index "ledgercreditvoidrecord_id" to table: "ledger_credit_void_records"
CREATE UNIQUE INDEX "ledgercreditvoidrecord_id" ON "ledger_credit_void_records" ("id");
-- create index "ledgercreditvoidrecord_namespace" to table: "ledger_credit_void_records"
CREATE INDEX "ledgercreditvoidrecord_namespace" ON "ledger_credit_void_records" ("namespace");
-- create index "ledgercreditvoidrecord_namespace_customer_currency_voided" to table: "ledger_credit_void_records"
CREATE INDEX "ledgercreditvoidrecord_namespace_customer_currency_voided" ON "ledger_credit_void_records" ("namespace", "customer_id", "currency", "voided_at", "id");
-- create index "ledgercreditvoidrecord_namespace_source_charge_id" to table: "ledger_credit_void_records"
CREATE INDEX "ledgercreditvoidrecord_namespace_source_charge_id" ON "ledger_credit_void_records" ("namespace", "source_charge_id");
-- create index "ledgercreditvoidrecord_namespace_void_transaction_group_id" to table: "ledger_credit_void_records"
CREATE INDEX "ledgercreditvoidrecord_namespace_void_transaction_group_id" ON "ledger_credit_void_records" ("namespace", "void_transaction_group_id");
