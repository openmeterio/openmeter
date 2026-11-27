-- Run after Rule type normalization and after all older assignment writers have drained.
BEGIN;

-- Capture only the live Channels locked by this transaction. New Channels already
-- start with an empty filter. Use the same lock order and strength as the writer.
CREATE TEMP TABLE notification_filter_backfill_channels ON COMMIT DROP AS
SELECT id, namespace
FROM notification_channels
WHERE deleted_at IS NULL
ORDER BY id
FOR NO KEY UPDATE;

-- A separate statement sees assignments committed while waiting for a Channel lock.
WITH filters AS (
    SELECT c.id,
           COALESCE(jsonb_agg(DISTINCT r.type COLLATE "C" ORDER BY r.type COLLATE "C")
                    FILTER (WHERE r.type IS NOT NULL), '[]'::jsonb) AS expected
    FROM pg_temp.notification_filter_backfill_channels c
    LEFT JOIN notification_channel_rules a ON a.notification_channel_id = c.id
    LEFT JOIN notification_rules r ON r.id = a.notification_rule_id
        AND r.namespace = c.namespace AND r.deleted_at IS NULL
    GROUP BY c.id
)
UPDATE notification_channels c
SET event_type_filter = f.expected
FROM filters f
WHERE c.id = f.id AND c.event_type_filter IS DISTINCT FROM f.expected;

COMMIT;
