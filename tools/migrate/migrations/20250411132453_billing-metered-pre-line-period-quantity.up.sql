-- modify "billing_invoice_usage_based_line_configs" table
ALTER TABLE "billing_invoice_usage_based_line_configs" ADD COLUMN "metered_pre_line_period_quantity" numeric NULL;

-- before usage discounts were introduced, we didn't have the metered pre-line period quantity, now let's backfill it
UPDATE "billing_invoice_usage_based_line_configs"
    SET "metered_pre_line_period_quantity" = "pre_line_period_quantity"
    WHERE
        "pre_line_period_quantity" IS NOT NULL AND
        "metered_pre_line_period_quantity" IS NULL;

-- before usage discounts were introduced, we didn't have the metered quantity, now let's backfill it
UPDATE "billing_invoice_usage_based_line_configs"
    SET "metered_quantity" = "billing_invoice_lines"."quantity"
    FROM "billing_invoice_lines"
    WHERE "billing_invoice_lines"."usage_based_line_config_id" = "billing_invoice_usage_based_line_configs"."id" AND
          "billing_invoice_lines"."type" = 'usage_based' AND
          "billing_invoice_lines"."quantity" IS NOT NULL AND
        "billing_invoice_usage_based_line_configs"."metered_quantity" IS NULL;
