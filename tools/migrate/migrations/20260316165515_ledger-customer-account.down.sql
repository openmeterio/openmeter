-- reverse: create index "ledgercustomeraccount_namespace_id" to table: "ledger_customer_accounts"
DROP INDEX "ledgercustomeraccount_namespace_id";
-- reverse: create index "ledgercustomeraccount_namespace_customer_id_account_type" to table: "ledger_customer_accounts"
DROP INDEX "ledgercustomeraccount_namespace_customer_id_account_type";
-- reverse: create index "ledgercustomeraccount_namespace" to table: "ledger_customer_accounts"
DROP INDEX "ledgercustomeraccount_namespace";
-- reverse: create index "ledgercustomeraccount_id" to table: "ledger_customer_accounts"
DROP INDEX "ledgercustomeraccount_id";
-- reverse: create "ledger_customer_accounts" table
DROP TABLE "ledger_customer_accounts";
