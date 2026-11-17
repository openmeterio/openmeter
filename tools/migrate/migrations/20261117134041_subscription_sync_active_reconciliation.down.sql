-- reverse: modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" DROP CONSTRAINT "subscription_sync_v2_active_reconciliation_at", DROP COLUMN "next_reconciliation_at";
