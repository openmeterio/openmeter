-- drop index "ledgerdimension_namespace_dimension_key_dimension_value" from table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace_dimension_key_dimension_value";
-- modify "ledger_dimensions" table
ALTER TABLE "ledger_dimensions" ADD COLUMN "dimension_display_value" character varying NOT NULL;
-- create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
-- create "ledger_sub_accounts" table
CREATE TABLE "ledger_sub_accounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "account_id" character(26) NOT NULL,
  "ledger_dimension_sub_accounts" character(26) NULL,
  "currency_dimension_id" character(26) NOT NULL,
  "tax_code_dimension_id" character(26) NULL,
  "features_dimension_id" character(26) NULL,
  "credit_priority_dimension_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ledger_sub_accounts_ledger_accounts_sub_accounts" FOREIGN KEY ("account_id") REFERENCES "ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_credit_priority_sub_accou" FOREIGN KEY ("credit_priority_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_currency_sub_accounts" FOREIGN KEY ("currency_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_features_sub_accounts" FOREIGN KEY ("features_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_sub_accounts" FOREIGN KEY ("ledger_dimension_sub_accounts") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_tax_code_sub_accounts" FOREIGN KEY ("tax_code_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "ledgersubaccount_annotations" to table: "ledger_sub_accounts"
CREATE INDEX "ledgersubaccount_annotations" ON "ledger_sub_accounts" USING gin ("annotations");
-- create index "ledgersubaccount_id" to table: "ledger_sub_accounts"
CREATE UNIQUE INDEX "ledgersubaccount_id" ON "ledger_sub_accounts" ("id");
-- create index "ledgersubaccount_namespace" to table: "ledger_sub_accounts"
CREATE INDEX "ledgersubaccount_namespace" ON "ledger_sub_accounts" ("namespace");
-- create index "ledgersubaccount_namespace_account_id_currency_dimension_id" to table: "ledger_sub_accounts"
CREATE UNIQUE INDEX "ledgersubaccount_namespace_account_id_currency_dimension_id" ON "ledger_sub_accounts" ("namespace", "account_id", "currency_dimension_id");
-- modify "ledger_entries" table
ALTER TABLE "ledger_entries" DROP COLUMN "account_id", DROP COLUMN "account_type", ADD COLUMN "sub_account_id" character(26) NOT NULL, ADD CONSTRAINT "ledger_entries_ledger_sub_accounts_entries" FOREIGN KEY ("sub_account_id") REFERENCES "ledger_sub_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- create index "ledgerentry_namespace_sub_account_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_sub_account_id" ON "ledger_entries" ("namespace", "sub_account_id");
