BEGIN;

CREATE INDEX om_mig_bil_fee_cfg_id_idx
    ON billing_invoice_lines (fee_line_config_id);
CREATE INDEX om_mig_bil_ubp_cfg_id_idx
    ON billing_invoice_lines (usage_based_line_config_id);
CREATE INDEX om_mig_bil_parent_id_idx
    ON billing_invoice_lines (parent_line_id);
CREATE INDEX om_mig_bild_line_id_idx
    ON billing_invoice_line_discounts (line_id);
CREATE INDEX om_mig_biuld_line_id_idx
    ON billing_invoice_line_usage_discounts (line_id);
CREATE INDEX om_mig_bsidl_parent_id_idx
    ON billing_standard_invoice_detailed_lines (parent_line_id);
CREATE INDEX om_mig_bsidlad_line_id_idx
    ON billing_standard_invoice_detailed_line_amount_discounts (line_id);

CREATE TABLE om_migration_backup_20260716150455_gathering_details (
    id VARCHAR NOT NULL,
    namespace VARCHAR NOT NULL,
    source_table TEXT NOT NULL,
    row_data JSONB NOT NULL,
    PRIMARY KEY (source_table, namespace, id)
);

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT dl.id, dl.namespace, 'billing_invoice_lines', to_jsonb(dl)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_invoice_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based'
    AND dl.status = 'detailed';

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT d.id, d.namespace, 'billing_invoice_line_discounts', to_jsonb(d)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_invoice_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
JOIN billing_invoice_line_discounts d
    ON d.line_id = dl.id
    AND d.namespace = dl.namespace
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based'
    AND dl.status = 'detailed';

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT d.id, d.namespace, 'billing_invoice_line_usage_discounts', to_jsonb(d)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_invoice_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
JOIN billing_invoice_line_usage_discounts d
    ON d.line_id = dl.id
    AND d.namespace = dl.namespace
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based'
    AND dl.status = 'detailed';

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT c.id, c.namespace, 'billing_invoice_flat_fee_line_configs', to_jsonb(c)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_invoice_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
JOIN billing_invoice_flat_fee_line_configs c
    ON c.id = dl.fee_line_config_id
    AND c.namespace = dl.namespace
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based'
    AND dl.status = 'detailed';

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT c.id, c.namespace, 'billing_invoice_usage_based_line_configs', to_jsonb(c)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_invoice_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
JOIN billing_invoice_usage_based_line_configs c
    ON c.id = dl.usage_based_line_config_id
    AND c.namespace = dl.namespace
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based'
    AND dl.status = 'detailed';

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT dl.id, dl.namespace, 'billing_standard_invoice_detailed_lines', to_jsonb(dl)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_standard_invoice_detailed_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based';

INSERT INTO om_migration_backup_20260716150455_gathering_details (id, namespace, source_table, row_data)
SELECT d.id, d.namespace, 'billing_standard_invoice_detailed_line_amount_discounts', to_jsonb(d)
FROM billing_invoices i
JOIN billing_invoice_lines l
    ON l.invoice_id = i.id
    AND l.namespace = i.namespace
JOIN billing_standard_invoice_detailed_lines dl
    ON dl.invoice_id = i.id
    AND dl.namespace = i.namespace
    AND dl.parent_line_id = l.id
JOIN billing_standard_invoice_detailed_line_amount_discounts d
    ON d.line_id = dl.id
    AND d.namespace = dl.namespace
WHERE
    i.status = 'gathering'
    AND l.parent_line_id IS NULL
    AND l.status = 'valid'
    AND l.type = 'usage_based';

DELETE FROM billing_standard_invoice_detailed_line_amount_discounts d
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_standard_invoice_detailed_line_amount_discounts'
    AND d.id = b.id
    AND d.namespace = b.namespace;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_standard_invoice_detailed_line_amount_discounts d
            ON d.line_id = b.id
            AND d.namespace = b.namespace
        WHERE b.source_table = 'billing_standard_invoice_detailed_lines'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice standard details would cascade to amount discounts';
    END IF;
END
$$;

DELETE FROM billing_standard_invoice_detailed_lines dl
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_standard_invoice_detailed_lines'
    AND dl.id = b.id
    AND dl.namespace = b.namespace;

DELETE FROM billing_invoice_line_discounts d
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_invoice_line_discounts'
    AND d.id = b.id
    AND d.namespace = b.namespace;

DELETE FROM billing_invoice_line_usage_discounts d
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_invoice_line_usage_discounts'
    AND d.id = b.id
    AND d.namespace = b.namespace;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_invoice_line_discounts d
            ON d.line_id = b.id
            AND d.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_lines'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice details would cascade to billing_invoice_line_discounts';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_invoice_line_usage_discounts d
            ON d.line_id = b.id
            AND d.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_lines'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice details would cascade to billing_invoice_line_usage_discounts';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_standard_invoice_detailed_lines dl
            ON dl.parent_line_id = b.id
            AND dl.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_lines'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice details would cascade to billing_standard_invoice_detailed_lines';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN charge_credit_purchase_invoiced_payments p
            ON p.line_id = b.id
            AND p.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_lines'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice details would cascade to charge_credit_purchase_invoiced_payments';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_invoice_lines dl
            ON dl.parent_line_id = b.id
            AND dl.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_lines'
    ) THEN
        RAISE EXCEPTION 'gathering invoice detail selected for deletion is not a leaf line';
    END IF;
END
$$;

DELETE FROM billing_invoice_lines dl
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_invoice_lines'
    AND dl.id = b.id
    AND dl.namespace = b.namespace;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_invoice_lines l
            ON l.fee_line_config_id = b.id
            AND l.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_flat_fee_line_configs'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice flat-fee configs would cascade to billing_invoice_lines';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM om_migration_backup_20260716150455_gathering_details b
        JOIN billing_invoice_lines l
            ON l.usage_based_line_config_id = b.id
            AND l.namespace = b.namespace
        WHERE b.source_table = 'billing_invoice_usage_based_line_configs'
    ) THEN
        RAISE EXCEPTION 'deleting gathering invoice usage-based configs would cascade to billing_invoice_lines';
    END IF;
END
$$;

DELETE FROM billing_invoice_flat_fee_line_configs c
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_invoice_flat_fee_line_configs'
    AND c.id = b.id
    AND c.namespace = b.namespace;

DELETE FROM billing_invoice_usage_based_line_configs c
USING om_migration_backup_20260716150455_gathering_details b
WHERE
    b.source_table = 'billing_invoice_usage_based_line_configs'
    AND c.id = b.id
    AND c.namespace = b.namespace;

DROP INDEX
    om_mig_bil_fee_cfg_id_idx,
    om_mig_bil_ubp_cfg_id_idx,
    om_mig_bil_parent_id_idx,
    om_mig_bild_line_id_idx,
    om_mig_biuld_line_id_idx,
    om_mig_bsidl_parent_id_idx,
    om_mig_bsidlad_line_id_idx;

COMMIT;
