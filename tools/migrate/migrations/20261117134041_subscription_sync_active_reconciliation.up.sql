-- modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" ADD CONSTRAINT "subscription_sync_v2_active_reconciliation_at" CHECK (((last_success_state)::text <> 'active'::text) OR (next_reconciliation_at IS NOT NULL)), ADD COLUMN "next_reconciliation_at" timestamptz NULL;
