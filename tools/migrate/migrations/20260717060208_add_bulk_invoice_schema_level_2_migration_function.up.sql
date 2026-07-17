-- Migrates schema-level-1 invoices for an arbitrary set of customers to schema level 2.
--
-- Gathering invoice details are intentionally excluded: they are invalid legacy data and are
-- removed by the preceding gathering-detail cleanup migration.
CREATE OR REPLACE FUNCTION om_func_migrate_customer_invoices_to_schema_level_2_bulk(p_customer_ids TEXT[])
RETURNS BIGINT
AS $$
DECLARE
    v_updated_invoices BIGINT;
BEGIN
    INSERT INTO billing_standard_invoice_detailed_lines (
        id,
        namespace,
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
        l.namespace,
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
        i.customer_id = ANY(p_customer_ids)
        AND i.schema_level = 1
        AND i.status <> 'gathering'
        AND l.status = 'detailed'
        AND l.type = 'flat_fee'
    ON CONFLICT (id) DO NOTHING;

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
    JOIN billing_standard_invoice_detailed_lines migrated_l
        ON migrated_l.id = l.id
        AND migrated_l.namespace = l.namespace
    JOIN billing_invoice_line_discounts d
        ON d.line_id = l.id
        AND d.namespace = l.namespace
    WHERE
        i.customer_id = ANY(p_customer_ids)
        AND i.schema_level = 1
        AND i.status <> 'gathering'
        AND l.status = 'detailed'
        AND l.type = 'flat_fee'
    ON CONFLICT (id) DO NOTHING;

    UPDATE billing_invoices
    SET schema_level = 2
    WHERE
        customer_id = ANY(p_customer_ids)
        AND schema_level = 1;

    GET DIAGNOSTICS v_updated_invoices = ROW_COUNT;

    RETURN v_updated_invoices;
END;
$$ LANGUAGE plpgsql VOLATILE;
