-- All APIs and workers must read both payload formats before the writer release.
-- Run after bare-payload writers are deployed everywhere and older writes drain.
CREATE FUNCTION pg_temp.notification_payload_to_snake_case(value jsonb) RETURNS jsonb
LANGUAGE plpgsql AS $$
DECLARE
    item record;
    mapped jsonb := '{}'::jsonb;
    mapped_value jsonb;
    mapped_key text;
BEGIN
    CASE jsonb_typeof(value)
    WHEN 'object' THEN
        FOR item IN SELECT key, val FROM jsonb_each(value) AS fields(key, val) LOOP
            mapped_key := lower(regexp_replace(regexp_replace(item.key,
                '([A-Z]+)([A-Z][a-z])', '\1_\2', 'g'), '([a-z0-9])([A-Z])', '\1_\2', 'g'));
            IF mapped ? mapped_key THEN
                RAISE EXCEPTION 'notification payload has conflicting property names for %', mapped_key;
            END IF;
            CASE item.key
            WHEN 'metadata', 'annotations', 'meterGroupByFilters' THEN
                mapped_value := item.val;
            WHEN 'advancedMeterGroupByFilters' THEN
                IF item.val = 'null'::jsonb THEN
                    mapped_value := item.val;
                ELSE
                    SELECT COALESCE(jsonb_object_agg(key, pg_temp.notification_payload_to_snake_case(val)), '{}'::jsonb)
                    INTO mapped_value FROM jsonb_each(item.val) AS filters(key, val);
                END IF;
            ELSE
                mapped_value := pg_temp.notification_payload_to_snake_case(item.val);
            END CASE;
            mapped := mapped || jsonb_build_object(mapped_key, mapped_value);
        END LOOP;
        RETURN mapped;
    WHEN 'array' THEN
        SELECT COALESCE(jsonb_agg(pg_temp.notification_payload_to_snake_case(val) ORDER BY ord), '[]'::jsonb)
        INTO mapped FROM jsonb_array_elements(value) WITH ORDINALITY AS items(val, ord);
        RETURN mapped;
    ELSE
        RETURN value;
    END CASE;
END;
$$;

DO $$
DECLARE
    event_row record;
BEGIN
    FOR event_row IN
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
    SELECT * FROM checked ORDER BY namespace, id
    LOOP
        IF event_row.problem IS NOT NULL THEN
            RAISE EXCEPTION 'invalid notification payload [namespace=% event.id=%]: %',
                event_row.namespace, event_row.id, event_row.problem;
        END IF;
        IF event_row.has_envelope THEN
            UPDATE notification_events
            SET payload = pg_temp.notification_payload_to_snake_case(event_row.payload->event_row.wrapper)
            WHERE namespace = event_row.namespace AND id = event_row.id;
        END IF;
    END LOOP;
END;
$$;

DROP FUNCTION pg_temp.notification_payload_to_snake_case(jsonb);
