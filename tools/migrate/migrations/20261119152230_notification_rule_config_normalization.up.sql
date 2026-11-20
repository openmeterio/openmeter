-- Apply once all writers of legacy Rule configs and their transactions have drained.
-- Retain the original envelope for recovery and leave already-flat configs alone.
DO $$
DECLARE
    rule RECORD;
BEGIN
    FOR rule IN
        SELECT r.id, r.namespace, r.config, names.legacy
        FROM notification_rules AS r
        JOIN (VALUES
            ('entitlements.balance.threshold', 'v1.entitlement.balance_threshold.triggered'),
            ('entitlements.reset', 'v1.entitlement.usage_period.reset'),
            ('invoice.created', 'v1.billing.invoice.created'),
            ('invoice.updated', 'v1.billing.invoice.updated')
        ) AS names(legacy, versioned) ON r.type IN (names.legacy, names.versioned)
        WHERE r.config ?| ARRAY['type', 'data']
        ORDER BY r.id
        FOR UPDATE OF r
    LOOP
        IF (rule.config->>'type' = rule.legacy AND jsonb_typeof(rule.config->'data') = 'object') IS NOT TRUE THEN
            RAISE EXCEPTION 'invalid rule config envelope [table=notification_rules namespace=% rule.id=%]: expected type "%" and an object-valued data field. '
                'Repair the discriminator and data, or supply the corresponding flat config. '
                'If schema_om shows version 20261119152230 as dirty, verify this migration rolled back and use golang-migrate with x-migrations-table=schema_om to force 20261119143941 (the previous successful version), then rerun the migration.', rule.namespace, rule.id, rule.legacy;
        END IF;

        UPDATE notification_rules
        SET config = rule.config->'data',
            -- Existing keys win so a second upgrade preserves the first backup.
            annotations = jsonb_build_object('dbmigration.notification.rule.config.normalization.original', rule.config)
                || COALESCE(NULLIF(annotations, 'null'::jsonb), '{}'::jsonb)
        WHERE id = rule.id AND namespace = rule.namespace;
    END LOOP;
END $$;
