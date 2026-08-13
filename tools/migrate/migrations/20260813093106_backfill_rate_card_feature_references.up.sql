-- Backfill incomplete feature references left by the former rate-card feature resolver.
-- Mark modified rows so the data change is attributable and reversible. Unresolved or
-- ambiguous references are intentionally left unchanged for external validation.
BEGIN;

-- ID-only references identify a feature directly. The namespace predicate prevents a
-- cross-namespace reference from being completed by the migration.
UPDATE plan_rate_cards AS rate_card
SET
  feature_key = feature.key,
  annotations = COALESCE(NULLIF(rate_card.annotations, 'null'::jsonb), '{}'::jsonb) ||
    jsonb_build_object(
      'dbmigration:backfill_rate_card_feature_references',
      jsonb_build_object(
        'field', 'feature_key',
        'at', to_char(
          CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
      )
    )
FROM features AS feature
WHERE rate_card.feature_id = feature.id
  AND rate_card.namespace = feature.namespace
  AND NULLIF(rate_card.feature_key, '') IS NULL
  AND NOT COALESCE(
    rate_card.annotations ? 'dbmigration:backfill_rate_card_feature_references',
    false
  );

UPDATE addon_rate_cards AS rate_card
SET
  feature_key = feature.key,
  annotations = COALESCE(NULLIF(rate_card.annotations, 'null'::jsonb), '{}'::jsonb) ||
    jsonb_build_object(
      'dbmigration:backfill_rate_card_feature_references',
      jsonb_build_object(
        'field', 'feature_key',
        'at', to_char(
          CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
      )
    )
FROM features AS feature
WHERE rate_card.feature_id = feature.id
  AND rate_card.namespace = feature.namespace
  AND NULLIF(rate_card.feature_key, '') IS NULL
  AND NOT COALESCE(
    rate_card.annotations ? 'dbmigration:backfill_rate_card_feature_references',
    false
  );

-- Key-only references use the feature version active at the rate card's historical
-- updated_at. Only a unique temporal match is repaired; zero or multiple matches are
-- intentionally skipped rather than failing the migration.
WITH resolved_features AS (
  SELECT rate_card.id, MAX(feature.id) AS feature_id
  FROM plan_rate_cards AS rate_card
  JOIN features AS feature
    ON feature.namespace = rate_card.namespace
   AND feature.key = rate_card.feature_key
   AND feature.created_at <= rate_card.updated_at
   AND (feature.archived_at IS NULL OR feature.archived_at > rate_card.updated_at)
   AND (feature.deleted_at IS NULL OR feature.deleted_at > rate_card.updated_at)
  WHERE rate_card.feature_id IS NULL
    AND NULLIF(rate_card.feature_key, '') IS NOT NULL
    AND NOT COALESCE(
      rate_card.annotations ? 'dbmigration:backfill_rate_card_feature_references',
      false
    )
  GROUP BY rate_card.id
  HAVING COUNT(*) = 1
)
UPDATE plan_rate_cards AS rate_card
SET
  feature_id = resolved_feature.feature_id,
  annotations = COALESCE(NULLIF(rate_card.annotations, 'null'::jsonb), '{}'::jsonb) ||
    jsonb_build_object(
      'dbmigration:backfill_rate_card_feature_references',
      jsonb_build_object(
        'field', 'feature_id',
        'at', to_char(
          CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
      )
    )
FROM resolved_features AS resolved_feature
WHERE rate_card.id = resolved_feature.id;

WITH resolved_features AS (
  SELECT rate_card.id, MAX(feature.id) AS feature_id
  FROM addon_rate_cards AS rate_card
  JOIN features AS feature
    ON feature.namespace = rate_card.namespace
   AND feature.key = rate_card.feature_key
   AND feature.created_at <= rate_card.updated_at
   AND (feature.archived_at IS NULL OR feature.archived_at > rate_card.updated_at)
   AND (feature.deleted_at IS NULL OR feature.deleted_at > rate_card.updated_at)
  WHERE rate_card.feature_id IS NULL
    AND NULLIF(rate_card.feature_key, '') IS NOT NULL
    AND NOT COALESCE(
      rate_card.annotations ? 'dbmigration:backfill_rate_card_feature_references',
      false
    )
  GROUP BY rate_card.id
  HAVING COUNT(*) = 1
)
UPDATE addon_rate_cards AS rate_card
SET
  feature_id = resolved_feature.feature_id,
  annotations = COALESCE(NULLIF(rate_card.annotations, 'null'::jsonb), '{}'::jsonb) ||
    jsonb_build_object(
      'dbmigration:backfill_rate_card_feature_references',
      jsonb_build_object(
        'field', 'feature_id',
        'at', to_char(
          CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
      )
    )
FROM resolved_features AS resolved_feature
WHERE rate_card.id = resolved_feature.id;

COMMIT;
