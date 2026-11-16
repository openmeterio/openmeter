-- modify "subscription_sync_states_v2" table
-- Migration 20261116101155 fills every NULL subscription revision before these constraints are applied.
-- atlas:nolint MF104
ALTER TABLE "subscription_sync_states_v2" ALTER COLUMN "last_synced_subs_revision" SET NOT NULL, ALTER COLUMN "current_subs_revision" SET NOT NULL;
