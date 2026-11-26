-- Rule configs are flat after the earlier config migration. Preserve threshold order,
-- other fields, and the Rule's revision and timestamps, including on deleted Rules.
UPDATE notification_rules AS r
SET config = jsonb_set(r.config, '{thresholds}', (
    SELECT jsonb_agg(
        CASE threshold->>'type'
            WHEN 'NUMBER' THEN jsonb_set(threshold, '{type}', '"usage_value"')
            WHEN 'PERCENT' THEN jsonb_set(threshold, '{type}', '"usage_percentage"')
            ELSE threshold
        END ORDER BY position
    )
    FROM jsonb_array_elements(r.config->'thresholds') WITH ORDINALITY AS t(threshold, position)
))
WHERE r.type IN ('entitlements.balance.threshold', 'v1.entitlement.balance_threshold.triggered')
  AND (r.config @> '{"thresholds":[{"type":"NUMBER"}]}' OR r.config @> '{"thresholds":[{"type":"PERCENT"}]}');
