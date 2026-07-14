-- reverse: create index "ledgercreditvoidrecord_namespace_void_transaction_group_id" to table: "ledger_credit_void_records"
DROP INDEX "ledgercreditvoidrecord_namespace_void_transaction_group_id";
-- reverse: create index "ledgercreditvoidrecord_namespace_source_charge_id" to table: "ledger_credit_void_records"
DROP INDEX "ledgercreditvoidrecord_namespace_source_charge_id";
-- reverse: create index "ledgercreditvoidrecord_namespace_customer_currency_voided" to table: "ledger_credit_void_records"
DROP INDEX "ledgercreditvoidrecord_namespace_customer_currency_voided";
-- reverse: create index "ledgercreditvoidrecord_namespace" to table: "ledger_credit_void_records"
DROP INDEX "ledgercreditvoidrecord_namespace";
-- reverse: create index "ledgercreditvoidrecord_id" to table: "ledger_credit_void_records"
DROP INDEX "ledgercreditvoidrecord_id";
-- reverse: create index "ledgercreditvoidrecord_annotations" to table: "ledger_credit_void_records"
DROP INDEX "ledgercreditvoidrecord_annotations";
-- reverse: create "ledger_credit_void_records" table
DROP TABLE "ledger_credit_void_records";
