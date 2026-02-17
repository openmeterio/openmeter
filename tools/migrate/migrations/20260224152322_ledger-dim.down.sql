-- reverse: create index "ledgerentry_namespace_sub_account_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_sub_account_id";
-- reverse: modify "ledger_entries" table
ALTER TABLE "ledger_entries" DROP CONSTRAINT "ledger_entries_ledger_sub_accounts_entries", DROP COLUMN "sub_account_id", ADD COLUMN "account_type" character varying NOT NULL, ADD COLUMN "account_id" character(26) NOT NULL;
-- reverse: create index "ledgersubaccount_namespace_account_id_currency_dimension_id" to table: "ledger_sub_accounts"
DROP INDEX "ledgersubaccount_namespace_account_id_currency_dimension_id";
-- reverse: create index "ledgersubaccount_namespace" to table: "ledger_sub_accounts"
DROP INDEX "ledgersubaccount_namespace";
-- reverse: create index "ledgersubaccount_id" to table: "ledger_sub_accounts"
DROP INDEX "ledgersubaccount_id";
-- reverse: create index "ledgersubaccount_annotations" to table: "ledger_sub_accounts"
DROP INDEX "ledgersubaccount_annotations";
-- reverse: create "ledger_sub_accounts" table
DROP TABLE "ledger_sub_accounts";
-- reverse: create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace_dimension_key_dimension_value";
-- reverse: modify "ledger_dimensions" table
ALTER TABLE "ledger_dimensions" DROP COLUMN "dimension_display_value";
-- reverse: drop index "ledgerdimension_namespace_dimension_key_dimension_value" from table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
