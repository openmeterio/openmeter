-- Deduplicate tax_codes rows that share the same Stripe app_mapping content.
--
-- Two write paths can produce duplicate (namespace, app_type, app_tax_code) groups:
-- the OSS seeder (system-managed keys like "saas_business") and the dual-write path
-- GetOrCreateByAppMapping (auto-keyed as "stripe_txcd_XXXXXXXX"). The read-side
-- tie-break already returns the right row, but duplicate rows remain on disk and
-- downstream tax_code_id FK columns may still point at losers. This migration
-- soft-deletes every loser and repoints all FKs to the winner in one transaction.

BEGIN;

-- Step 1: materialise loser → winner pairs into a temp table.
-- Winner selection mirrors the read-side tie-break: system-managed rows first,
-- then oldest created_at, then smallest id.
DROP TABLE IF EXISTS _tax_code_dedup_map;
CREATE TEMP TABLE _tax_code_dedup_map AS
WITH expanded AS (
  SELECT
    t.id,
    t.namespace,
    t.created_at,
    (m->>'app_type') AS app_type,
    (m->>'tax_code') AS app_tax_code,
    COALESCE(t.annotations->>'managed_by', '') = 'system' AS is_system
  FROM tax_codes t,
       LATERAL jsonb_array_elements(COALESCE(t.app_mappings, '[]'::jsonb)) AS m
  WHERE t.deleted_at IS NULL
    AND t.app_mappings IS NOT NULL
),
ranked AS (
  SELECT *,
    ROW_NUMBER() OVER (
      PARTITION BY namespace, app_type, app_tax_code
      ORDER BY is_system DESC, created_at ASC, id ASC
    ) AS rn,
    COUNT(*) OVER (PARTITION BY namespace, app_type, app_tax_code) AS group_size
  FROM expanded
),
winners AS (
  SELECT namespace, app_type, app_tax_code, id AS winner_id
  FROM ranked WHERE rn = 1 AND group_size > 1
)
SELECT DISTINCT r.id AS loser_id, w.winner_id
FROM ranked r
JOIN winners w USING (namespace, app_type, app_tax_code)
WHERE r.rn > 1;

-- Sanity check: a single loser_id must map to exactly one winner_id. A
-- multi-mapping tax_codes row can be ranked as a loser in two partitions
-- whose winners differ; the subsequent UPDATE ... FROM _tax_code_dedup_map
-- would then non-deterministically pick one winner per child row. Abort
-- fast instead so a human can inspect the data.
DO $$
DECLARE
  conflicting_count int;
BEGIN
  SELECT COUNT(*) INTO conflicting_count
  FROM (
    SELECT loser_id
    FROM _tax_code_dedup_map
    GROUP BY loser_id
    HAVING COUNT(DISTINCT winner_id) > 1
  ) c;

  IF conflicting_count > 0 THEN
    RAISE EXCEPTION 'tax_code dedup: % loser row(s) map to multiple distinct winner rows; aborting migration', conflicting_count;
  END IF;
END $$;

CREATE INDEX ON _tax_code_dedup_map (loser_id);

-- Step 2: repoint every tax_code_id FK column to the winner.
-- Tables that carry a paired JSONB column (productcatalog.TaxConfig) also have
-- the embedded tax_code_id scalar repointed in the same UPDATE so that
-- both representations stay consistent without a second table pass.
-- Tables without a JSONB column (charge_flat_fees, charge_usage_based,
-- charge_credit_purchases, organization_default_tax_codes) only update the FK.
UPDATE billing_workflow_configs
   SET tax_code_id                  = m.winner_id,
       invoice_default_tax_settings = jsonb_set(invoice_default_tax_settings, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE billing_workflow_configs.tax_code_id = m.loser_id;

UPDATE billing_customer_overrides
   SET tax_code_id                = m.winner_id,
       invoice_default_tax_config = jsonb_set(invoice_default_tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE billing_customer_overrides.tax_code_id = m.loser_id;

UPDATE billing_invoice_lines
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE billing_invoice_lines.tax_code_id = m.loser_id;

UPDATE billing_invoice_split_line_groups
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE billing_invoice_split_line_groups.tax_code_id = m.loser_id;

UPDATE billing_standard_invoice_detailed_lines
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE billing_standard_invoice_detailed_lines.tax_code_id = m.loser_id;

UPDATE charge_usage_based_run_detailed_line
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE charge_usage_based_run_detailed_line.tax_code_id = m.loser_id;

UPDATE charge_flat_fee_run_detailed_lines
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE charge_flat_fee_run_detailed_lines.tax_code_id = m.loser_id;

UPDATE subscription_items
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE subscription_items.tax_code_id = m.loser_id;

UPDATE plan_rate_cards
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE plan_rate_cards.tax_code_id = m.loser_id;

UPDATE addon_rate_cards
   SET tax_code_id = m.winner_id,
       tax_config  = jsonb_set(tax_config, '{tax_code_id}', to_jsonb(m.winner_id::text))
   FROM _tax_code_dedup_map m
  WHERE addon_rate_cards.tax_code_id = m.loser_id;

UPDATE charge_flat_fees                         SET tax_code_id = m.winner_id FROM _tax_code_dedup_map m WHERE charge_flat_fees.tax_code_id                        = m.loser_id;
UPDATE charge_usage_based                       SET tax_code_id = m.winner_id FROM _tax_code_dedup_map m WHERE charge_usage_based.tax_code_id                      = m.loser_id;
UPDATE charge_credit_purchases                  SET tax_code_id = m.winner_id FROM _tax_code_dedup_map m WHERE charge_credit_purchases.tax_code_id                 = m.loser_id;
UPDATE organization_default_tax_codes           SET invoicing_tax_code_id    = m.winner_id FROM _tax_code_dedup_map m WHERE organization_default_tax_codes.invoicing_tax_code_id    = m.loser_id;
UPDATE organization_default_tax_codes           SET credit_grant_tax_code_id = m.winner_id FROM _tax_code_dedup_map m WHERE organization_default_tax_codes.credit_grant_tax_code_id = m.loser_id;

-- Step 3: rewrite the full taxcode.TaxCode entity snapshot embedded inside
-- billing_invoice_lines.tax_config.tax_code. SnapshotTaxConfigIntoLines
-- (openmeter/billing/service/invoicecalc/taxconfig.go) stamps the resolved
-- entity into this sub-object at invoice-advance time; without rewriting
-- it, raw-SQL or edge-less reads would still see the loser's id, key, name.
--
-- Match on the loser id still embedded inside the snapshot so only lines
-- whose snapshot is actually stale get rewritten. The guard
-- `tax_config -> 'tax_code' IS NOT NULL` skips gathering invoices that
-- never had a snapshot stamped.
UPDATE billing_invoice_lines bil
   SET tax_config = jsonb_set(
         bil.tax_config,
         '{tax_code}',
         jsonb_build_object(
           'namespace',    w.namespace,
           'id',           w.id,
           'createdAt',    w.created_at,
           'updatedAt',    w.updated_at,
           'deletedAt',    w.deleted_at,
           'key',          w.key,
           'name',         w.name,
           'description',  w.description,
           'app_mappings', COALESCE(w.app_mappings::jsonb, '[]'::jsonb),
           'metadata',     COALESCE(w.metadata,            '{}'::jsonb),
           'annotations',  COALESCE(w.annotations,         '{}'::jsonb)
         )
       )
   FROM _tax_code_dedup_map m
   JOIN tax_codes w ON w.id = m.winner_id
  WHERE bil.tax_config -> 'tax_code' IS NOT NULL
    AND bil.tax_config -> 'tax_code' ->> 'id' = m.loser_id;

-- Step 4: soft-delete losers.
UPDATE tax_codes
  SET deleted_at = NOW(), updated_at = NOW()
  WHERE id IN (SELECT loser_id FROM _tax_code_dedup_map)
    AND deleted_at IS NULL;

-- Step 5: drop the temp table explicitly. Without ON COMMIT DROP (which atlas
-- migrate validate cannot model under its autocommit dry-run), session-scoped
-- temp tables would otherwise live until the migration connection closes.
DROP TABLE _tax_code_dedup_map;

COMMIT;
