BEGIN;

-- Deploy only after the schema-level-2 backfill and all application instances have
-- stopped using the retired storage. Keep the archive and cleanup in one transaction.
DO $$
DECLARE
    tables_to_lock TEXT;
BEGIN
    -- Stabilize the target rows and every table whose foreign keys could otherwise
    -- cascade, detach, or acquire new references while we archive and delete them.
    SELECT string_agg(table_id::regclass::text, ', ' ORDER BY table_id)
    INTO tables_to_lock
    FROM (
        SELECT conrelid AS table_id
        FROM pg_constraint
        WHERE contype = 'f'
          AND confrelid IN ('billing_invoice_lines'::regclass, 'billing_invoice_usage_based_line_configs'::regclass)
        UNION
        SELECT unnest(ARRAY[
            'billing_invoices'::regclass,
            'billing_invoice_lines'::regclass,
            'billing_gathering_invoice_lines'::regclass,
            'billing_invoice_flat_fee_line_configs'::regclass,
            'billing_invoice_usage_based_line_configs'::regclass,
            'billing_invoice_line_discounts'::regclass,
            'billing_invoice_line_usage_discounts'::regclass
        ])::oid
    ) tables;
    EXECUTE 'LOCK TABLE ' || tables_to_lock || ' IN SHARE ROW EXCLUSIVE MODE';

    -- Empty invoices do not contain any data requiring backfill.
    IF EXISTS (
        SELECT 1 FROM billing_invoices i
        WHERE i.schema_level <> 2
          AND (
              EXISTS (SELECT 1 FROM billing_invoice_lines l WHERE l.invoice_id = i.id)
              OR EXISTS (SELECT 1 FROM billing_gathering_invoice_lines l WHERE l.invoice_id = i.id)
          )
    ) THEN
        RAISE EXCEPTION 'legacy detailed line cleanup requires schema level 2 for invoices with lines';
    END IF;

    IF EXISTS (
        SELECT 1 FROM billing_invoice_lines
        WHERE status <> 'detailed' AND fee_line_config_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'surviving invoice lines still reference legacy flat-fee configurations';
    END IF;
END
$$;

-- Select by storage status alone: missing parents/configurations and soft deletion
-- must not leave unarchived legacy data behind.
CREATE TEMP TABLE om_legacy_detailed_line_ids ON COMMIT DROP AS
SELECT id, usage_based_line_config_id
FROM billing_invoice_lines
WHERE status = 'detailed';
ALTER TABLE om_legacy_detailed_line_ids ADD PRIMARY KEY (id);

DO $$
DECLARE
    reference RECORD;
    has_surviving_reference BOOLEAN;
BEGIN
    -- Discover consumers from the database, including newer charge tables. Both
    -- CASCADE and SET NULL would silently change surviving data without this guard.
    FOR reference IN
        SELECT c.conrelid, c.conname, c.conkey, c.confkey, a.attname
        FROM pg_constraint c
        JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
        WHERE c.contype = 'f' AND c.confrelid = 'billing_invoice_lines'::regclass
          AND c.conrelid NOT IN ('billing_invoice_line_discounts'::regclass, 'billing_invoice_line_usage_discounts'::regclass)
    LOOP
        IF cardinality(reference.conkey) <> 1 OR cardinality(reference.confkey) <> 1
            OR reference.confkey[1] <> (SELECT attnum FROM pg_attribute WHERE attrelid = 'billing_invoice_lines'::regclass AND attname = 'id') THEN
            RAISE EXCEPTION 'unsupported invoice-line foreign key %', reference.conname;
        END IF;
        EXECUTE format(
            'SELECT EXISTS (SELECT 1 FROM %s r JOIN pg_temp.om_legacy_detailed_line_ids t ON r.%I = t.id %s)',
            reference.conrelid::regclass, reference.attname,
            CASE WHEN reference.conrelid = 'billing_invoice_lines'::regclass
                THEN 'WHERE NOT EXISTS (SELECT 1 FROM pg_temp.om_legacy_detailed_line_ids child WHERE child.id = r.id)'
                ELSE '' END
        ) INTO has_surviving_reference;
        IF has_surviving_reference THEN
            RAISE EXCEPTION 'legacy detailed lines are still referenced by %.%', reference.conrelid::regclass, reference.attname;
        END IF;
    END LOOP;
END
$$;

CREATE TABLE IF NOT EXISTS om_migration_backup_legacy_detailed_lines (
    backed_up_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    id VARCHAR NOT NULL,
    namespace VARCHAR NOT NULL,
    source_table TEXT NOT NULL,
    row_data JSONB NOT NULL,
    PRIMARY KEY (backed_up_at, source_table, namespace, id)
);

INSERT INTO om_migration_backup_legacy_detailed_lines (id, namespace, source_table, row_data)
SELECT l.id, l.namespace, 'billing_invoice_lines', jsonb_build_object(
    'line', to_jsonb(l),
    'flat_fee_config', (SELECT to_jsonb(c) FROM billing_invoice_flat_fee_line_configs c WHERE c.id = l.fee_line_config_id),
    'usage_based_config', (SELECT to_jsonb(c) FROM billing_invoice_usage_based_line_configs c WHERE c.id = l.usage_based_line_config_id),
    'amount_discounts', COALESCE((
        SELECT jsonb_agg(to_jsonb(d) ORDER BY d.id) FROM billing_invoice_line_discounts d WHERE d.line_id = l.id
    ), '[]'::jsonb),
    'usage_discounts', COALESCE((
        SELECT jsonb_agg(to_jsonb(d) ORDER BY d.id) FROM billing_invoice_line_usage_discounts d WHERE d.line_id = l.id
    ), '[]'::jsonb)
)
FROM billing_invoice_lines l
JOIN om_legacy_detailed_line_ids t ON t.id = l.id;

-- Retired tables may also contain unreferenced rows, or old amount discounts
-- attached to surviving standard lines. Archive those independently before DROP.
INSERT INTO om_migration_backup_legacy_detailed_lines (id, namespace, source_table, row_data)
SELECT c.id, c.namespace, 'billing_invoice_flat_fee_line_configs', jsonb_build_object('flat_fee_config', to_jsonb(c))
FROM billing_invoice_flat_fee_line_configs c
WHERE NOT EXISTS (
    SELECT 1 FROM billing_invoice_lines l JOIN om_legacy_detailed_line_ids t ON t.id = l.id
    WHERE l.fee_line_config_id = c.id
);

INSERT INTO om_migration_backup_legacy_detailed_lines (id, namespace, source_table, row_data)
SELECT d.id, d.namespace, 'billing_invoice_line_discounts', jsonb_build_object('amount_discount', to_jsonb(d))
FROM billing_invoice_line_discounts d
WHERE NOT EXISTS (SELECT 1 FROM om_legacy_detailed_line_ids t WHERE t.id = d.line_id);

DO $$
DECLARE
    uncovered_table TEXT;
BEGIN
    -- Compare full original rows with their JSON representations before deleting
    -- anything. Separate child arrays avoid multiplying rows when both have data.
    WITH backup AS (
        SELECT source_table, row_data FROM om_migration_backup_legacy_detailed_lines
        WHERE backed_up_at = transaction_timestamp()
    ), expected AS (
        SELECT 'billing_invoice_lines' AS source_table, to_jsonb(l) AS row_data
        FROM billing_invoice_lines l JOIN om_legacy_detailed_line_ids t ON t.id = l.id
        UNION ALL
        SELECT 'billing_invoice_flat_fee_line_configs', to_jsonb(c) FROM billing_invoice_flat_fee_line_configs c
        UNION ALL
        SELECT 'billing_invoice_line_discounts', to_jsonb(d) FROM billing_invoice_line_discounts d
        UNION ALL
        SELECT 'billing_invoice_line_usage_discounts', to_jsonb(d)
        FROM billing_invoice_line_usage_discounts d JOIN om_legacy_detailed_line_ids t ON t.id = d.line_id
        UNION ALL
        SELECT 'billing_invoice_usage_based_line_configs', to_jsonb(c)
        FROM billing_invoice_usage_based_line_configs c
        WHERE EXISTS (SELECT 1 FROM om_legacy_detailed_line_ids t WHERE t.usage_based_line_config_id = c.id)
    ), archived AS (
        SELECT 'billing_invoice_lines' AS source_table, row_data -> 'line' AS row_data
        FROM backup WHERE source_table = 'billing_invoice_lines'
        UNION ALL
        SELECT 'billing_invoice_flat_fee_line_configs', row_data -> 'flat_fee_config' FROM backup
        UNION ALL
        SELECT 'billing_invoice_usage_based_line_configs', row_data -> 'usage_based_config' FROM backup
        UNION ALL
        SELECT 'billing_invoice_line_discounts', d.value
        FROM backup b, jsonb_array_elements(b.row_data -> 'amount_discounts') d
        UNION ALL
        SELECT 'billing_invoice_line_discounts', row_data -> 'amount_discount'
        FROM backup WHERE source_table = 'billing_invoice_line_discounts'
        UNION ALL
        SELECT 'billing_invoice_line_usage_discounts', d.value
        FROM backup b, jsonb_array_elements(b.row_data -> 'usage_discounts') d
    )
    SELECT source_table INTO uncovered_table
    FROM (SELECT * FROM expected EXCEPT SELECT * FROM archived) missing
    LIMIT 1;
    IF uncovered_table IS NOT NULL THEN
        RAISE EXCEPTION 'legacy detailed line backup is incomplete for %', uncovered_table;
    END IF;
END
$$;

DELETE FROM billing_invoice_line_usage_discounts d
USING om_legacy_detailed_line_ids t WHERE d.line_id = t.id;
DELETE FROM billing_invoice_line_discounts d
USING om_legacy_detailed_line_ids t WHERE d.line_id = t.id;

-- Break only links internal to the archived set before self-referential FK actions
-- run. Surviving split-line relationships were checked above and remain untouched.
UPDATE billing_invoice_lines l SET parent_line_id = NULL
FROM om_legacy_detailed_line_ids t
WHERE l.id = t.id AND l.parent_line_id IN (SELECT id FROM om_legacy_detailed_line_ids);
DELETE FROM billing_invoice_lines l USING om_legacy_detailed_line_ids t WHERE l.id = t.id;

-- The usage configuration table is still live. Shared configurations must survive.
DELETE FROM billing_invoice_usage_based_line_configs c
WHERE EXISTS (SELECT 1 FROM om_legacy_detailed_line_ids t WHERE t.usage_based_line_config_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM billing_invoice_lines l WHERE l.usage_based_line_config_id = c.id);

DROP FUNCTION om_func_migrate_customer_invoices_to_schema_level_2(TEXT);
DROP FUNCTION om_func_migrate_customer_invoices_to_schema_level_2_bulk(TEXT[]);

-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "fee_line_config_id";
-- drop "billing_invoice_flat_fee_line_configs" table
DROP TABLE "billing_invoice_flat_fee_line_configs";
-- drop "billing_invoice_line_discounts" table
DROP TABLE "billing_invoice_line_discounts";

-- Build recovery indexes after loading the archive to avoid maintaining them for
-- every inserted row. Keep the retained archive usable across schema round trips.
CREATE INDEX IF NOT EXISTS om_legacy_detail_backup_id_idx
    ON om_migration_backup_legacy_detailed_lines (id);
CREATE INDEX IF NOT EXISTS om_legacy_detail_backup_ns_invoice_idx
    ON om_migration_backup_legacy_detailed_lines (namespace, ((row_data -> 'line') ->> 'invoice_id'));
CREATE INDEX IF NOT EXISTS om_legacy_detail_backup_ns_parent_idx
    ON om_migration_backup_legacy_detailed_lines (namespace, ((row_data -> 'line') ->> 'parent_line_id'));
CREATE INDEX IF NOT EXISTS om_legacy_detail_backup_data_idx
    ON om_migration_backup_legacy_detailed_lines USING gin (row_data jsonb_path_ops);
ANALYZE om_migration_backup_legacy_detailed_lines;

COMMIT;
