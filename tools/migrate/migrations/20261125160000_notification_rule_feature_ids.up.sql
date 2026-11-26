-- Deploy the ID-resolving Rule writer everywhere and drain older writers before
-- applying this migration. Include archived features and soft-deleted Rules:
-- historical references must not silently move to a replacement feature.
DO $$
DECLARE
    rule RECORD;
    reference JSONB;
    feature_id TEXT;
    match_count BIGINT;
    resolved JSONB;
BEGIN
    FOR rule IN
        SELECT r.id, r.namespace, r.config, r.created_at, r.updated_at, r.deleted_at
        FROM notification_rules AS r
        WHERE r.type IN (
            'entitlements.balance.threshold', 'v1.entitlement.balance_threshold.triggered',
            'entitlements.reset', 'v1.entitlement.usage_period.reset'
        )
        ORDER BY r.id
        FOR UPDATE OF r
    LOOP
        IF jsonb_typeof(rule.config) IS DISTINCT FROM 'object' OR rule.config ?| ARRAY['type', 'data'] THEN
            RAISE EXCEPTION 'expected normalized flat rule config [namespace=% rule.id=%]', rule.namespace, rule.id;
        END IF;

        -- V4 accepts an omitted or empty list for unrestricted scope, but rejects JSON null.
        IF rule.config->'features' = 'null'::jsonb THEN
            UPDATE notification_rules SET config = config - 'features' WHERE id = rule.id;
            CONTINUE;
        END IF;

        IF NOT rule.config ? 'features' THEN
            CONTINUE;
        END IF;

        IF jsonb_typeof(rule.config->'features') IS DISTINCT FROM 'array' THEN
            RAISE EXCEPTION 'invalid rule features [namespace=% rule.id=%]: expected an array', rule.namespace, rule.id;
        END IF;

        resolved := '[]'::jsonb;
        FOR reference IN SELECT value FROM jsonb_array_elements(rule.config->'features')
        LOOP
            IF NOT (
                (jsonb_typeof(reference) = 'string' AND reference #>> '{}' <> '')
                OR (reference = jsonb_build_object('id', reference->>'id')
                    AND COALESCE(reference->>'id', '') ~ '^[0-7][0-9A-HJKMNP-TV-Z]{25}$')
            ) THEN
                RAISE EXCEPTION 'invalid rule feature reference [namespace=% rule.id=%]: expected a nonempty string or an ID reference', rule.namespace, rule.id;
            END IF;

            -- Soft deletion advances updated_at without changing the config. For
            -- deleted Rules, any feature active during the Rule's life is a candidate.
            SELECT count(*), min(f.id::text) INTO match_count, feature_id
            FROM features AS f
            WHERE f.namespace = rule.namespace
              AND (
                  f.id::text = COALESCE(reference->>'id', reference #>> '{}')
                  OR (
                      jsonb_typeof(reference) = 'string' AND f.key = reference #>> '{}'
                      AND CASE WHEN rule.deleted_at IS NULL THEN f.created_at <= rule.updated_at
                               ELSE f.created_at < rule.deleted_at END
                      AND (f.archived_at IS NULL OR f.archived_at >
                           CASE WHEN rule.deleted_at IS NULL THEN rule.updated_at ELSE rule.created_at END)
                      AND (f.deleted_at IS NULL OR f.deleted_at >
                           CASE WHEN rule.deleted_at IS NULL THEN rule.updated_at ELSE rule.created_at END)
                  )
              );

            IF match_count = 0 THEN
                CONTINUE;
            END IF;

            IF match_count > 1 THEN
                RAISE EXCEPTION 'ambiguous rule feature [namespace=% rule.id=% reference=% matches=%]: repair the reference and rerun; no Rule configs were changed',
                    rule.namespace, rule.id, reference, match_count;
            END IF;

            -- Preserve first-occurrence order when both an ID and its key occur.
            IF NOT resolved @> jsonb_build_array(jsonb_build_object('id', feature_id)) THEN
                resolved := resolved || jsonb_build_array(jsonb_build_object('id', feature_id));
            END IF;
        END LOOP;

        IF resolved IS DISTINCT FROM rule.config->'features' THEN
            UPDATE notification_rules
            SET config = jsonb_set(rule.config, '{features}', resolved)
            WHERE id = rule.id AND namespace = rule.namespace;
        END IF;
    END LOOP;
END $$;
