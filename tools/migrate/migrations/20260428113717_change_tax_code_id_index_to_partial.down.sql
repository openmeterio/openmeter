-- reverse: create index "chargeusagebased_tax_code_id" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_tax_code_id";
-- reverse: drop index "chargeusagebased_tax_code_id" from table: "charge_usage_based"
CREATE INDEX "chargeusagebased_tax_code_id" ON "charge_usage_based" ("tax_code_id");
-- reverse: create index "chargeflatfee_tax_code_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_tax_code_id";
-- reverse: drop index "chargeflatfee_tax_code_id" from table: "charge_flat_fees"
CREATE INDEX "chargeflatfee_tax_code_id" ON "charge_flat_fees" ("tax_code_id");
-- reverse: create index "chargecreditpurchase_tax_code_id" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_tax_code_id";
-- reverse: drop index "chargecreditpurchase_tax_code_id" from table: "charge_credit_purchases"
CREATE INDEX "chargecreditpurchase_tax_code_id" ON "charge_credit_purchases" ("tax_code_id");
