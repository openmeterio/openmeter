-- reverse: create index "ledgerentry_namespace_transaction_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_transaction_id";
-- reverse: create index "ledgerentry_namespace_sub_account_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_sub_account_id";
-- reverse: create index "ledgerentry_namespace_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_id";
-- reverse: create index "ledgerentry_namespace" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace";
-- reverse: create index "ledgerentry_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_id";
-- reverse: create index "ledgerentry_created_at_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_created_at_id";
-- reverse: create index "ledgerentry_annotations" to table: "ledger_entries"
DROP INDEX "ledgerentry_annotations";
-- reverse: create "ledger_entries" table
DROP TABLE "ledger_entries";
-- reverse: create index "ledgertransaction_namespace_id" to table: "ledger_transactions"
DROP INDEX "ledgertransaction_namespace_id";
-- reverse: create index "ledgertransaction_namespace_group_id" to table: "ledger_transactions"
DROP INDEX "ledgertransaction_namespace_group_id";
-- reverse: create index "ledgertransaction_namespace_booked_at" to table: "ledger_transactions"
DROP INDEX "ledgertransaction_namespace_booked_at";
-- reverse: create index "ledgertransaction_namespace" to table: "ledger_transactions"
DROP INDEX "ledgertransaction_namespace";
-- reverse: create index "ledgertransaction_id" to table: "ledger_transactions"
DROP INDEX "ledgertransaction_id";
-- reverse: create index "ledgertransaction_annotations" to table: "ledger_transactions"
DROP INDEX "ledgertransaction_annotations";
-- reverse: create "ledger_transactions" table
DROP TABLE "ledger_transactions";
-- reverse: create index "ledgertransactiongroup_namespace_id" to table: "ledger_transaction_groups"
DROP INDEX "ledgertransactiongroup_namespace_id";
-- reverse: create index "ledgertransactiongroup_namespace" to table: "ledger_transaction_groups"
DROP INDEX "ledgertransactiongroup_namespace";
-- reverse: create index "ledgertransactiongroup_id" to table: "ledger_transaction_groups"
DROP INDEX "ledgertransactiongroup_id";
-- reverse: create index "ledgertransactiongroup_annotations" to table: "ledger_transaction_groups"
DROP INDEX "ledgertransactiongroup_annotations";
-- reverse: create "ledger_transaction_groups" table
DROP TABLE "ledger_transaction_groups";
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
-- reverse: create index "ledgerdimension_namespace_id" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace_id";
-- reverse: create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace_dimension_key_dimension_value";
-- reverse: create index "ledgerdimension_namespace" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace";
-- reverse: create index "ledgerdimension_id" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_id";
-- reverse: create index "ledgerdimension_annotations" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_annotations";
-- reverse: create "ledger_dimensions" table
DROP TABLE "ledger_dimensions";
-- reverse: create index "ledgeraccount_namespace_id" to table: "ledger_accounts"
DROP INDEX "ledgeraccount_namespace_id";
-- reverse: create index "ledgeraccount_namespace" to table: "ledger_accounts"
DROP INDEX "ledgeraccount_namespace";
-- reverse: create index "ledgeraccount_id" to table: "ledger_accounts"
DROP INDEX "ledgeraccount_id";
-- reverse: create index "ledgeraccount_annotations" to table: "ledger_accounts"
DROP INDEX "ledgeraccount_annotations";
-- reverse: create "ledger_accounts" table
DROP TABLE "ledger_accounts";
