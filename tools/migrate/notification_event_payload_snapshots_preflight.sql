-- Read-only. Run after all bare-payload writers are deployed and older writes drain.
-- Unknown Event types are outside this migration's scope.
WITH event_types(legacy_type, canonical_type, wrapper, required_fields) AS (
    VALUES
        ('entitlements.balance.threshold', 'v1.entitlement.balance_threshold.triggered', 'balanceThreshold', ARRAY['entitlement', 'feature', 'subject', 'value', 'threshold']),
        ('entitlements.reset', 'v1.entitlement.usage_period.reset', 'entitlementReset', ARRAY['entitlement', 'feature', 'subject', 'value']),
        ('invoice.created', 'v1.billing.invoice.created', 'invoice', ARRAY['id']),
        ('invoice.updated', 'v1.billing.invoice.updated', 'invoice', ARRAY['id'])
), candidates AS (
    SELECT e.namespace, e.id, e.type, e.payload, t.*,
        (e.payload ? t.wrapper OR COALESCE(e.payload->>'type' IN (
            'entitlements.balance.threshold', 'entitlements.reset', 'invoice.created', 'invoice.updated'
        ), false)) AS has_envelope
    FROM notification_events e
    JOIN event_types t ON e.type IN (t.legacy_type, t.canonical_type)
), checked AS (
    SELECT *, CASE
        WHEN jsonb_typeof(payload) IS DISTINCT FROM 'object' THEN 'payload must be an object'
        WHEN has_envelope AND payload->>'type' IS DISTINCT FROM legacy_type THEN 'envelope type differs from Event type'
        WHEN has_envelope AND wrapper = 'invoice' AND payload->'version' IS DISTINCT FROM '1'::jsonb THEN 'unsupported invoice envelope version'
        WHEN has_envelope AND jsonb_typeof(payload->wrapper) IS DISTINCT FROM 'object' THEN 'missing or invalid envelope snapshot'
        WHEN has_envelope AND payload - ARRAY['type', 'version', wrapper] <> '{}'::jsonb THEN 'unexpected envelope fields require review'
        WHEN EXISTS (
            SELECT 1 FROM unnest(required_fields) field
            WHERE NOT (CASE WHEN has_envelope THEN payload->wrapper ELSE payload END) ? field
               OR (CASE WHEN has_envelope THEN payload->wrapper ELSE payload END)->field = 'null'::jsonb
        ) THEN 'snapshot is missing required fields'
    END AS problem
    FROM candidates
)
SELECT namespace, id AS event_id, type AS event_type,
    CASE WHEN problem IS NOT NULL THEN 'invalid'
         WHEN has_envelope THEN 'legacy_envelope'
         ELSE 'already_canonical' END AS status,
    problem
FROM checked
ORDER BY namespace, id;
