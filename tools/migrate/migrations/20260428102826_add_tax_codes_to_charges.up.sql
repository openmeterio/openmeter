-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
