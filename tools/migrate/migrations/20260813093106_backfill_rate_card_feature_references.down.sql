BEGIN;

UPDATE plan_rate_cards
SET
  feature_key = NULL,
  annotations = CASE
    WHEN annotations - 'dbmigration:backfill_rate_card_feature_references' = '{}'::jsonb
      THEN NULL
    ELSE annotations - 'dbmigration:backfill_rate_card_feature_references'
  END
WHERE annotations -> 'dbmigration:backfill_rate_card_feature_references' ->> 'field' = 'feature_key';

UPDATE plan_rate_cards
SET
  feature_id = NULL,
  annotations = CASE
    WHEN annotations - 'dbmigration:backfill_rate_card_feature_references' = '{}'::jsonb
      THEN NULL
    ELSE annotations - 'dbmigration:backfill_rate_card_feature_references'
  END
WHERE annotations -> 'dbmigration:backfill_rate_card_feature_references' ->> 'field' = 'feature_id';

UPDATE addon_rate_cards
SET
  feature_key = NULL,
  annotations = CASE
    WHEN annotations - 'dbmigration:backfill_rate_card_feature_references' = '{}'::jsonb
      THEN NULL
    ELSE annotations - 'dbmigration:backfill_rate_card_feature_references'
  END
WHERE annotations -> 'dbmigration:backfill_rate_card_feature_references' ->> 'field' = 'feature_key';

UPDATE addon_rate_cards
SET
  feature_id = NULL,
  annotations = CASE
    WHEN annotations - 'dbmigration:backfill_rate_card_feature_references' = '{}'::jsonb
      THEN NULL
    ELSE annotations - 'dbmigration:backfill_rate_card_feature_references'
  END
WHERE annotations -> 'dbmigration:backfill_rate_card_feature_references' ->> 'field' = 'feature_id';

COMMIT;
