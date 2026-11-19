-- Restore legacy type names, including Rules and Events created after upgrade.
-- Stop and drain versioned-name writers before rolling back to older readers.
-- JSON configs, payloads and their internal type discriminators stay unchanged.
BEGIN;

UPDATE notification_rules
SET type = CASE type
    WHEN 'v1.entitlement.balance_threshold.triggered' THEN 'entitlements.balance.threshold'
    WHEN 'v1.entitlement.usage_period.reset' THEN 'entitlements.reset'
    WHEN 'v1.billing.invoice.created' THEN 'invoice.created'
    WHEN 'v1.billing.invoice.updated' THEN 'invoice.updated'
END
WHERE type IN (
    'v1.entitlement.balance_threshold.triggered',
    'v1.entitlement.usage_period.reset',
    'v1.billing.invoice.created',
    'v1.billing.invoice.updated'
);

UPDATE notification_events
SET type = CASE type
    WHEN 'v1.entitlement.balance_threshold.triggered' THEN 'entitlements.balance.threshold'
    WHEN 'v1.entitlement.usage_period.reset' THEN 'entitlements.reset'
    WHEN 'v1.billing.invoice.created' THEN 'invoice.created'
    WHEN 'v1.billing.invoice.updated' THEN 'invoice.updated'
END
WHERE type IN (
    'v1.entitlement.balance_threshold.triggered',
    'v1.entitlement.usage_period.reset',
    'v1.billing.invoice.created',
    'v1.billing.invoice.updated'
);

COMMIT;
