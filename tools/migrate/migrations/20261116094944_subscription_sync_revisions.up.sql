-- modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" ADD COLUMN "last_synced_subs_revision" character(26) NULL, ADD COLUMN "current_subs_revision" character(26) NULL;
