-- Generate one revision per row. Preserve existing revisions, including any
-- difference between the subscription's current and last synced revisions.
WITH revisions AS MATERIALIZED (
  SELECT id, om_func_generate_ulid() AS revision
  FROM subscription_sync_states_v2
  WHERE last_synced_subs_revision IS NULL OR current_subs_revision IS NULL
)
UPDATE subscription_sync_states_v2 AS state
SET last_synced_subs_revision = COALESCE(state.last_synced_subs_revision, revisions.revision),
    current_subs_revision = COALESCE(state.current_subs_revision, revisions.revision)
FROM revisions
WHERE state.id = revisions.id;
