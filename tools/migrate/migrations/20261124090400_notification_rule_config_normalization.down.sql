-- Drain flat-config writers before rollback. Wrap the current config to preserve
-- Rules created or edited since the upgrade. Rollback leaves annotations untouched.
UPDATE notification_rules AS r
SET config = jsonb_build_object('type', names.legacy, 'data', r.config)
FROM (VALUES
    ('entitlements.balance.threshold', 'v1.entitlement.balance_threshold.triggered'),
    ('entitlements.reset', 'v1.entitlement.usage_period.reset'),
    ('invoice.created', 'v1.billing.invoice.created'),
    ('invoice.updated', 'v1.billing.invoice.updated')
) AS names(legacy, versioned)
WHERE r.type IN (names.legacy, names.versioned)
  AND jsonb_typeof(r.config) = 'object'
  AND NOT (r.config ?| ARRAY['type', 'data']);
