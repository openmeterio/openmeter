-- Normalize Rule and Event type columns after all backends and workers write
-- versioned names and transactions started by older writers have completed.
-- JSON configs, payloads and their internal type discriminators stay unchanged.
BEGIN;

UPDATE notification_rules
SET type = CASE type
    WHEN 'entitlements.balance.threshold' THEN 'v1.entitlement.balance_threshold.triggered'
    WHEN 'entitlements.reset' THEN 'v1.entitlement.usage_period.reset'
    WHEN 'invoice.created' THEN 'v1.billing.invoice.created'
    WHEN 'invoice.updated' THEN 'v1.billing.invoice.updated'
END
WHERE type IN (
    'entitlements.balance.threshold',
    'entitlements.reset',
    'invoice.created',
    'invoice.updated'
);

UPDATE notification_events
SET type = CASE type
    WHEN 'entitlements.balance.threshold' THEN 'v1.entitlement.balance_threshold.triggered'
    WHEN 'entitlements.reset' THEN 'v1.entitlement.usage_period.reset'
    WHEN 'invoice.created' THEN 'v1.billing.invoice.created'
    WHEN 'invoice.updated' THEN 'v1.billing.invoice.updated'
END
WHERE type IN (
    'entitlements.balance.threshold',
    'entitlements.reset',
    'invoice.created',
    'invoice.updated'
);

COMMIT;
