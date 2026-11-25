-- modify "rte_credit_balances" table
ALTER TABLE "rte_credit_balances" ADD COLUMN "checkpoints" jsonb NULL;
-- modify "rte_metered_entitlement_balances" table
ALTER TABLE "rte_metered_entitlement_balances" ADD COLUMN "checkpoints" jsonb NULL;
