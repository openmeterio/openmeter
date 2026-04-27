-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "chargecreditpurchase_tax_code_id" to table: "charge_credit_purchases"
CREATE INDEX "chargecreditpurchase_tax_code_id" ON "charge_credit_purchases" ("tax_code_id");
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "chargeflatfee_tax_code_id" to table: "charge_flat_fees"
CREATE INDEX "chargeflatfee_tax_code_id" ON "charge_flat_fees" ("tax_code_id");
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "chargeusagebased_tax_code_id" to table: "charge_usage_based"
CREATE INDEX "chargeusagebased_tax_code_id" ON "charge_usage_based" ("tax_code_id");
