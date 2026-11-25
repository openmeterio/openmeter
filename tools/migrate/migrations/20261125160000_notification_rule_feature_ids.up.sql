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
        SELECT r.id, r.namespace, r.config, r.updated_at
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

        -- Preserve the existing representation of unrestricted feature scope.
        IF NOT rule.config ? 'features' OR rule.config->'features' = 'null'::jsonb THEN
            CONTINUE;
        END IF;

        IF jsonb_typeof(rule.config->'features') IS DISTINCT FROM 'array' THEN
            RAISE EXCEPTION 'invalid rule features [namespace=% rule.id=%]: expected an array', rule.namespace, rule.id;
        END IF;

        resolved := '[]'::jsonb;
        FOR reference IN SELECT value FROM jsonb_array_elements(rule.config->'features')
        LOOP
            IF jsonb_typeof(reference) IS DISTINCT FROM 'string' OR reference #>> '{}' = '' THEN
                RAISE EXCEPTION 'invalid rule feature reference [namespace=% rule.id=%]: expected a nonempty string', rule.namespace, rule.id;
            END IF;

            -- IDs already pin a feature. Resolve keys against the feature lifetime
            -- at the Rule's last update, not the currently active replacement.
            SELECT count(*), min(f.id::text) INTO match_count, feature_id
            FROM features AS f
            WHERE f.namespace = rule.namespace
              AND (
                  f.id::text = reference #>> '{}'
                  OR (
                      f.key = reference #>> '{}'
                      AND f.created_at <= rule.updated_at
                      AND (f.archived_at IS NULL OR f.archived_at > rule.updated_at)
                      AND (f.deleted_at IS NULL OR f.deleted_at > rule.updated_at)
                  )
              );

            IF match_count = 0 THEN
                CONTINUE;
            END IF;

            IF match_count > 1 THEN
                RAISE EXCEPTION 'ambiguous rule feature [namespace=% rule.id=% reference=% matches=%]: repair the reference and rerun; no Rule configs were changed',
                    rule.namespace, rule.id, reference #>> '{}', match_count;
            END IF;

            -- Preserve first-occurrence order when both an ID and its key occur.
            IF NOT resolved ? feature_id THEN
                resolved := resolved || jsonb_build_array(feature_id);
            END IF;
        END LOOP;

        IF resolved IS DISTINCT FROM rule.config->'features' THEN
            UPDATE notification_rules
            SET config = jsonb_set(rule.config, '{features}', resolved)
            WHERE id = rule.id AND namespace = rule.namespace;
        END IF;
    END LOOP;
END $$;
