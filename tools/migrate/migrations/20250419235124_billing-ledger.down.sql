-- reverse: create index "billingsubledgertransaction_subledger_id" to table: "billing_subledger_transactions"
DROP INDEX "billingsubledgertransaction_subledger_id";
-- reverse: create index "billingsubledgertransaction_namespace_id" to table: "billing_subledger_transactions"
DROP INDEX "billingsubledgertransaction_namespace_id";
-- reverse: create index "billingsubledgertransaction_namespace" to table: "billing_subledger_transactions"
DROP INDEX "billingsubledgertransaction_namespace";
-- reverse: create index "billingsubledgertransaction_ledger_id" to table: "billing_subledger_transactions"
DROP INDEX "billingsubledgertransaction_ledger_id";
-- reverse: create index "billingsubledgertransaction_id" to table: "billing_subledger_transactions"
DROP INDEX "billingsubledgertransaction_id";
-- reverse: create "billing_subledger_transactions" table
DROP TABLE "billing_subledger_transactions";
-- reverse: create index "billingsubledger_namespace_ledger_id_key" to table: "billing_subledgers"
DROP INDEX "billingsubledger_namespace_ledger_id_key";
-- reverse: create index "billingsubledger_namespace_id" to table: "billing_subledgers"
DROP INDEX "billingsubledger_namespace_id";
-- reverse: create index "billingsubledger_namespace" to table: "billing_subledgers"
DROP INDEX "billingsubledger_namespace";
-- reverse: create index "billingsubledger_id" to table: "billing_subledgers"
DROP INDEX "billingsubledger_id";
-- reverse: create "billing_subledgers" table
DROP TABLE "billing_subledgers";
-- reverse: create index "billingledger_namespace_customer_id_currency" to table: "billing_ledgers"
DROP INDEX "billingledger_namespace_customer_id_currency";
-- reverse: create index "billingledger_namespace" to table: "billing_ledgers"
DROP INDEX "billingledger_namespace";
-- reverse: create index "billingledger_id" to table: "billing_ledgers"
DROP INDEX "billingledger_id";
-- reverse: create "billing_ledgers" table
DROP TABLE "billing_ledgers";
