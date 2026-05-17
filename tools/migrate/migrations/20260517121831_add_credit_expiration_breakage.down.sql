-- reverse: create index "ledgerbreakagerecord_namespace_source_transaction_group_id" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace_source_transaction_group_id";
-- reverse: create index "ledgerbreakagerecord_namespace_source_entry_id" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace_source_entry_id";
-- reverse: create index "ledgerbreakagerecord_namespace_plan_id" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace_plan_id";
-- reverse: create index "ledgerbreakagerecord_namespace_customer_id_currency_credit_" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace_customer_id_currency_credit_";
-- reverse: create index "ledgerbreakagerecord_namespace_breakage_transaction_group_id" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace_breakage_transaction_group_id";
-- reverse: create index "ledgerbreakagerecord_namespace" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace";
-- reverse: create index "ledgerbreakagerecord_id" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_id";
-- reverse: create index "ledgerbreakagerecord_annotations" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_annotations";
-- reverse: create "ledger_breakage_records" table
DROP TABLE "ledger_breakage_records";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "expires_at";
