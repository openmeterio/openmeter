BEGIN;

WITH marked_rate_cards AS (
  SELECT
    id,
    annotations -> 'dbmigration:backfill_rate_card_feature_references' AS marker
  FROM plan_rate_cards
  WHERE annotations ? 'dbmigration:backfill_rate_card_feature_references'
)
UPDATE plan_rate_cards AS rate_card
SET
  feature_key = CASE
    WHEN marked.marker ->> 'field' = 'feature_key'
      AND marked.marker ->> 'rate_card_id' = rate_card.id
      AND marked.marker ->> 'feature_id' = rate_card.feature_id
      AND marked.marker ->> 'feature_key' = rate_card.feature_key
      THEN NULL
    ELSE rate_card.feature_key
  END,
  feature_id = CASE
    WHEN marked.marker ->> 'field' = 'feature_id'
      AND marked.marker ->> 'rate_card_id' = rate_card.id
      AND marked.marker ->> 'feature_id' = rate_card.feature_id
      AND marked.marker ->> 'feature_key' = rate_card.feature_key
      THEN NULL
    ELSE rate_card.feature_id
  END,
  annotations = CASE
    WHEN rate_card.annotations - 'dbmigration:backfill_rate_card_feature_references' = '{}'::jsonb
      THEN NULL
    ELSE rate_card.annotations - 'dbmigration:backfill_rate_card_feature_references'
  END
FROM marked_rate_cards AS marked
WHERE rate_card.id = marked.id;

WITH marked_rate_cards AS (
  SELECT
    id,
    annotations -> 'dbmigration:backfill_rate_card_feature_references' AS marker
  FROM addon_rate_cards
  WHERE annotations ? 'dbmigration:backfill_rate_card_feature_references'
)
UPDATE addon_rate_cards AS rate_card
SET
  feature_key = CASE
    WHEN marked.marker ->> 'field' = 'feature_key'
      AND marked.marker ->> 'rate_card_id' = rate_card.id
      AND marked.marker ->> 'feature_id' = rate_card.feature_id
      AND marked.marker ->> 'feature_key' = rate_card.feature_key
      THEN NULL
    ELSE rate_card.feature_key
  END,
  feature_id = CASE
    WHEN marked.marker ->> 'field' = 'feature_id'
      AND marked.marker ->> 'rate_card_id' = rate_card.id
      AND marked.marker ->> 'feature_id' = rate_card.feature_id
      AND marked.marker ->> 'feature_key' = rate_card.feature_key
      THEN NULL
    ELSE rate_card.feature_id
  END,
  annotations = CASE
    WHEN rate_card.annotations - 'dbmigration:backfill_rate_card_feature_references' = '{}'::jsonb
      THEN NULL
    ELSE rate_card.annotations - 'dbmigration:backfill_rate_card_feature_references'
  END
FROM marked_rate_cards AS marked
WHERE rate_card.id = marked.id;

COMMIT;
