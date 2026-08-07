-- create index "chargeflatfees_cost_basis_id" to table: "charge_flat_fees"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargeflatfees_cost_basis_id" ON "charge_flat_fees" ("cost_basis_id");
-- create index "chargeusagebased_cost_basis_id" to table: "charge_usage_based"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargeusagebased_cost_basis_id" ON "charge_usage_based" ("cost_basis_id");
-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchase_cost_basis_charge_fk", ADD CONSTRAINT "charge_credit_purchase_cost_basis_charge_fk" FOREIGN KEY ("cost_basis_id") REFERENCES "charge_credit_purchase_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "chargecreditpurchases_cost_basis_id" to table: "charge_credit_purchases"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargecreditpurchases_cost_basis_id" ON "charge_credit_purchases" ("cost_basis_id");
