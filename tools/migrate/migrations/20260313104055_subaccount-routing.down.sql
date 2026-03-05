-- reverse: create index "ledgersubaccount_namespace_account_id_route_id" to table: "ledger_sub_accounts"
DROP INDEX "ledgersubaccount_namespace_account_id_route_id";
-- reverse: modify "ledger_sub_accounts" table
ALTER TABLE "ledger_sub_accounts" DROP CONSTRAINT "ledger_sub_accounts_ledger_sub_account_routes_sub_accounts", DROP COLUMN "route_id", ADD COLUMN "credit_priority_dimension_id" character(26) NULL, ADD COLUMN "features_dimension_id" character(26) NULL, ADD COLUMN "tax_code_dimension_id" character(26) NULL, ADD COLUMN "currency_dimension_id" character(26) NOT NULL, ADD COLUMN "ledger_dimension_sub_accounts" character(26) NULL;
-- reverse: create index "ledgersubaccountroute_namespace_account_id_routing_key_version_" to table: "ledger_sub_account_routes"
DROP INDEX "ledgersubaccountroute_namespace_account_id_routing_key_version_";
-- reverse: create index "ledgersubaccountroute_namespace" to table: "ledger_sub_account_routes"
DROP INDEX "ledgersubaccountroute_namespace";
-- reverse: create index "ledgersubaccountroute_id" to table: "ledger_sub_account_routes"
DROP INDEX "ledgersubaccountroute_id";
-- reverse: create "ledger_sub_account_routes" table
DROP TABLE "ledger_sub_account_routes";
