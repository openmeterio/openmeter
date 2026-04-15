-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "line_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_runs_billing_invoice_lines_charge_usage_base" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "charge_usage_based_runs_line_id_key" to table: "charge_usage_based_runs"
CREATE UNIQUE INDEX "charge_usage_based_runs_line_id_key" ON "charge_usage_based_runs" ("line_id");
