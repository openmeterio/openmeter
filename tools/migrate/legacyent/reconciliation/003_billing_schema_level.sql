-- Frozen reconciliation for Ent baseline 20260709134422.
INSERT INTO "billing_invoice_write_schema_levels" ("id", "schema_level")
VALUES ('write_schema_level', 1)
ON CONFLICT ("id") DO UPDATE SET "schema_level" = EXCLUDED."schema_level";

CREATE OR REPLACE FUNCTION om_func_migrate_customer_invoices_to_schema_level_2(p_customer_id TEXT)
RETURNS BIGINT
AS $$
DECLARE
    v_updated_invoices BIGINT;
BEGIN
    -- 1) Copy detailed lines (schema level 1) into the schema-level-2 structure.
    INSERT INTO billing_standard_invoice_detailed_lines (
        id,
        annotations,
        namespace,
        metadata,
        created_at,
        updated_at,
        deleted_at,
        name,
        description,
        currency,
        amount,
        taxes_total,
        taxes_inclusive_total,
        taxes_exclusive_total,
        charges_total,
        discounts_total,
        total,
        service_period_start,
        service_period_end,
        quantity,
        invoicing_app_external_id,
        child_unique_reference_id,
        per_unit_amount,
        category,
        payment_term,
        index,
        invoice_id,
        parent_line_id,
        credits_total,
        credits_applied
    )
    SELECT
        l.id,
        l.annotations,
        l.namespace,
        l.metadata,
        l.created_at,
        l.updated_at,
        l.deleted_at,
        l.name,
        l.description,
        l.currency,
        l.amount,
        l.taxes_total,
        l.taxes_inclusive_total,
        l.taxes_exclusive_total,
        l.charges_total,
        l.discounts_total,
        l.total,
        l.period_start AS service_period_start,
        l.period_end AS service_period_end,
        l.quantity,
        l.invoicing_app_external_id,
        l.child_unique_reference_id,
        c.per_unit_amount,
        c.category,
        c.payment_term,
        c.index,
        l.invoice_id,
        l.parent_line_id,
        l.credits_total,
        l.credits_applied
    FROM billing_invoices i
    JOIN billing_invoice_lines l
        ON l.invoice_id = i.id
        AND l.namespace = i.namespace
    JOIN billing_invoice_flat_fee_line_configs c
        ON c.id = l.fee_line_config_id
        AND c.namespace = l.namespace
    WHERE
        i.customer_id = p_customer_id
        AND i.schema_level = 1
        AND l.status = 'detailed'
        AND l.type = 'flat_fee'
    ON CONFLICT (id) DO NOTHING;

    -- 2) Copy detailed line amount discounts into the schema-level-2 structure.
    INSERT INTO billing_standard_invoice_detailed_line_amount_discounts (
        id,
        namespace,
        created_at,
        updated_at,
        deleted_at,
        child_unique_reference_id,
        description,
        reason,
        invoicing_app_external_id,
        amount,
        rounding_amount,
        source_discount,
        line_id
    )
    SELECT
        d.id,
        d.namespace,
        d.created_at,
        d.updated_at,
        d.deleted_at,
        d.child_unique_reference_id,
        d.description,
        d.reason,
        d.invoicing_app_external_id,
        d.amount,
        d.rounding_amount,
        d.source_discount,
        d.line_id
    FROM billing_invoices i
    JOIN billing_invoice_lines l
        ON l.invoice_id = i.id
        AND l.namespace = i.namespace
    JOIN billing_invoice_line_discounts d
        ON d.line_id = l.id
        AND d.namespace = l.namespace
    WHERE
        i.customer_id = p_customer_id
        AND i.schema_level = 1
        AND l.status = 'detailed'
        AND l.type = 'flat_fee'
    ON CONFLICT (id) DO NOTHING;

    -- 3) Mark the invoices as migrated (schema_level = 2).
    UPDATE billing_invoices
    SET schema_level = 2
    WHERE
        customer_id = p_customer_id
        AND schema_level = 1;

    GET DIAGNOSTICS v_updated_invoices = ROW_COUNT;

    RETURN v_updated_invoices;
END;
$$ LANGUAGE plpgsql VOLATILE;
