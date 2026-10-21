-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD CONSTRAINT "billing_invoice_lines_billing_invoice_flat_fee_line_configs_fla" FOREIGN KEY ("fee_line_config_id") REFERENCES "billing_invoice_flat_fee_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
