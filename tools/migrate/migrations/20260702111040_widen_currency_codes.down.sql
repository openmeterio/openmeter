-- reverse: modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "credit_realization_lineages" table
ALTER TABLE "credit_realization_lineages" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ALTER COLUMN "currency" TYPE character varying(3);
