-- Run before the backfill to preview changes and again before enabling V4 routing.
WITH filters AS (
    SELECT c.id, c.namespace, c.disabled, c.event_type_filter,
           COALESCE(jsonb_agg(DISTINCT r.type COLLATE "C" ORDER BY r.type COLLATE "C")
                    FILTER (WHERE r.type IS NOT NULL), '[]'::jsonb) AS expected
    FROM notification_channels c
    LEFT JOIN notification_channel_rules a ON a.notification_channel_id = c.id
    LEFT JOIN notification_rules r ON r.id = a.notification_rule_id
        AND r.namespace = c.namespace AND r.deleted_at IS NULL
    WHERE c.deleted_at IS NULL
    GROUP BY c.id
)
SELECT namespace, id AS channel_id, event_type_filter, expected,
       event_type_filter IS DISTINCT FROM expected AS needs_backfill,
       disabled IS NOT TRUE AND expected = '[]'::jsonb AS would_become_catch_all
FROM filters
WHERE event_type_filter IS DISTINCT FROM expected OR (disabled IS NOT TRUE AND expected = '[]'::jsonb)
ORDER BY namespace, id;
