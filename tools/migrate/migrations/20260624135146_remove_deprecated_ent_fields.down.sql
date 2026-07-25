-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ADD COLUMN "tax_code_id" character(26) NULL, ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_config" jsonb NULL;
ALTER TABLE "charge_usage_based_run_detailed_line" ADD CONSTRAINT "charge_usage_based_run_detailed_line_tax_codes_charge_usage_bas" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
CREATE INDEX "chargeusagebasedrundetailedline_tax_code_id" ON "charge_usage_based_run_detailed_line" ("tax_code_id");
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "override_intent_deleted_at" timestamptz NULL, ADD COLUMN "override_present" boolean NOT NULL DEFAULT false, ADD COLUMN "override_billing_period_to" timestamptz NULL, ADD COLUMN "override_billing_period_from" timestamptz NULL, ADD COLUMN "override_full_service_period_to" timestamptz NULL, ADD COLUMN "override_full_service_period_from" timestamptz NULL, ADD COLUMN "override_service_period_to" timestamptz NULL, ADD COLUMN "override_service_period_from" timestamptz NULL, ADD COLUMN "override_tax_code_id" character(26) NULL, ADD COLUMN "override_tax_behavior" character varying NULL, ADD COLUMN "override_metadata" jsonb NULL, ADD COLUMN "override_description" character varying NULL, ADD COLUMN "override_name" character varying NULL, ADD COLUMN "override_discounts" jsonb NULL, ADD COLUMN "override_price" jsonb NULL, ADD COLUMN "override_feature_key" character varying NULL;
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "override_intent_deleted_at" timestamptz NULL, ADD COLUMN "override_present" boolean NOT NULL DEFAULT false, ADD COLUMN "override_billing_period_to" timestamptz NULL, ADD COLUMN "override_billing_period_from" timestamptz NULL, ADD COLUMN "override_full_service_period_to" timestamptz NULL, ADD COLUMN "override_full_service_period_from" timestamptz NULL, ADD COLUMN "override_service_period_to" timestamptz NULL, ADD COLUMN "override_service_period_from" timestamptz NULL, ADD COLUMN "override_tax_code_id" character(26) NULL, ADD COLUMN "override_tax_behavior" character varying NULL, ADD COLUMN "override_metadata" jsonb NULL, ADD COLUMN "override_description" character varying NULL, ADD COLUMN "override_name" character varying NULL, ADD COLUMN "override_percentage_discounts" jsonb NULL, ADD COLUMN "override_amount_before_proration" numeric NULL, ADD COLUMN "override_pro_rating" jsonb NULL, ADD COLUMN "override_payment_term" character varying NULL, ADD COLUMN "override_feature_key" character varying NULL;
-- reverse: modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" ADD COLUMN "tax_code_id" character(26) NULL, ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_config" jsonb NULL;
ALTER TABLE "charge_flat_fee_run_detailed_lines" ADD CONSTRAINT "charge_flat_fee_run_detailed_lines_tax_codes_charge_flat_fee_ru" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
CREATE INDEX "chargeflatfeerundetailedline_tax_code_id" ON "charge_flat_fee_run_detailed_lines" ("tax_code_id");
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ADD COLUMN "tax_code_id" character(26) NULL, ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_config" jsonb NULL;
ALTER TABLE "billing_standard_invoice_detailed_lines" ADD CONSTRAINT "billing_standard_invoice_detailed_lines_tax_codes_billing_stand" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
CREATE INDEX "billingstandardinvoicedetailedline_tax_code_id" ON "billing_standard_invoice_detailed_lines" ("tax_code_id");
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD COLUMN "tax_code_id" character(26) NULL, ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_config" jsonb NULL;
ALTER TABLE "billing_invoice_split_line_groups" ADD CONSTRAINT "billing_invoice_split_line_groups_tax_codes_billing_invoice_spl" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
CREATE INDEX "billinginvoicesplitlinegroup_tax_code_id" ON "billing_invoice_split_line_groups" ("tax_code_id");
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "line_ids" character(26) NULL;
-- reverse: modify "billing_invoice_line_discounts" table
ALTER TABLE "billing_invoice_line_discounts" ADD COLUMN "pre_line_period_quantity" numeric NULL, ADD COLUMN "quantity" numeric NULL, ADD COLUMN "type" character varying NULL;

-- restore schema-level migration function for the schema with deprecated detailed-line tax columns
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
        tax_config,
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
        l.tax_config,
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
