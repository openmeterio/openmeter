-- reverse: create index "chargecreditpurchases_cost_basis_id" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchases_cost_basis_id";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchase_cost_basis_charge_fk", ADD CONSTRAINT "charge_credit_purchase_cost_basis_charge_fk" FOREIGN KEY ("cost_basis_id") REFERENCES "charge_credit_purchase_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: create index "chargeusagebased_cost_basis_id" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_cost_basis_id";
-- reverse: create index "chargeflatfees_cost_basis_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfees_cost_basis_id";
