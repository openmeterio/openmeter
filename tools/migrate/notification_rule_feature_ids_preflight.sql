-- Run after Rule config normalization. The drop_* actions are informational.
-- Other findings block the migration. An empty feature list matches every feature.
WITH rules AS (
    SELECT namespace, id AS rule_id, updated_at, config->'features' AS refs,
           CASE
               WHEN jsonb_typeof(config) IS DISTINCT FROM 'object' OR config ?| ARRAY['type', 'data']
                   THEN 'expected_flat_config'
               WHEN jsonb_typeof(config->'features') NOT IN ('array', 'null')
                   THEN 'expected_feature_array'
           END AS issue
    FROM notification_rules
    WHERE type IN (
        'entitlements.balance.threshold', 'v1.entitlement.balance_threshold.triggered',
        'entitlements.reset', 'v1.entitlement.usage_period.reset'
    )
), checked AS (
    SELECT r.namespace, r.rule_id, r.updated_at, item.ref,
           matches.n, matches.ids,
           CASE
               WHEN jsonb_typeof(item.ref) <> 'string' OR item.ref #>> '{}' = '' THEN 'invalid_reference'
               WHEN matches.n = 0 THEN 'drop_missing_feature'
               WHEN matches.n > 1 THEN 'ambiguous_feature'
           END AS issue
    FROM rules r
    CROSS JOIN LATERAL jsonb_array_elements(
        CASE WHEN r.issue IS NULL AND jsonb_typeof(r.refs) = 'array' THEN r.refs ELSE '[]'::jsonb END
    ) AS item(ref)
    CROSS JOIN LATERAL (
        SELECT count(*) AS n, array_agg(f.id::text ORDER BY f.id) AS ids
        FROM features f
        WHERE f.namespace = r.namespace AND jsonb_typeof(item.ref) = 'string'
          AND (f.id::text = item.ref #>> '{}' OR (
              f.key = item.ref #>> '{}'
              AND f.created_at <= r.updated_at
              AND (f.archived_at IS NULL OR f.archived_at > r.updated_at)
              AND (f.deleted_at IS NULL OR f.deleted_at > r.updated_at)
          ))
    ) matches
), actions AS (
    SELECT checked.*,
           count(*) FILTER (WHERE issue IS DISTINCT FROM 'drop_missing_feature')
               OVER (PARTITION BY namespace, rule_id) = 0 AS all_missing
    FROM checked
)
SELECT namespace, rule_id, updated_at, issue, NULL::jsonb AS reference,
       NULL::bigint AS match_count, NULL::text[] AS candidate_ids
FROM rules WHERE issue IS NOT NULL
UNION ALL
SELECT namespace, rule_id, updated_at,
       CASE WHEN all_missing THEN 'drop_missing_feature_filter_becomes_catch_all' ELSE issue END,
       ref, n, ids
FROM actions WHERE issue IS NOT NULL
ORDER BY namespace, rule_id, reference;
