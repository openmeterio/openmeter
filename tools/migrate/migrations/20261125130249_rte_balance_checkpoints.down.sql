-- reverse: modify "rte_metered_entitlement_balances" table
ALTER TABLE "rte_metered_entitlement_balances" DROP COLUMN "checkpoints";
-- reverse: modify "rte_credit_balances" table
ALTER TABLE "rte_credit_balances" DROP COLUMN "checkpoints";
