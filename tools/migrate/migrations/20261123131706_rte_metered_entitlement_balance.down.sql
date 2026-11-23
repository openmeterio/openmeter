-- reverse: create index "rtemeteredentitlementbalance_namespace" to table: "rte_metered_entitlement_balances"
DROP INDEX "rtemeteredentitlementbalance_namespace";
-- reverse: create index "rte_metered_balance_scope" to table: "rte_metered_entitlement_balances"
DROP INDEX "rte_metered_balance_scope";
-- reverse: create "rte_metered_entitlement_balances" table
DROP TABLE "rte_metered_entitlement_balances";
