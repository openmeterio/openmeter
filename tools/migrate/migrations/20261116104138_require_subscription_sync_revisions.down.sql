-- reverse: modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" ALTER COLUMN "current_subs_revision" DROP NOT NULL, ALTER COLUMN "last_synced_subs_revision" DROP NOT NULL;
