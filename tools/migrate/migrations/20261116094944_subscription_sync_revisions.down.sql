-- reverse: modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" DROP COLUMN "current_subs_revision", DROP COLUMN "last_synced_subs_revision";
