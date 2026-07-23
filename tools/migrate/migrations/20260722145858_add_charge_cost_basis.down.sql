-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_cost_basis_charge_fk", DROP COLUMN "cost_basis_id";
-- reverse: create index "chargeusagebasedcostbasis_resolved_cost_basis_id" to table: "charge_usage_based_cost_bases"
DROP INDEX "chargeusagebasedcostbasis_resolved_cost_basis_id";
-- reverse: create index "chargeusagebasedcostbasis_namespace" to table: "charge_usage_based_cost_bases"
DROP INDEX "chargeusagebasedcostbasis_namespace";
-- reverse: create index "chargeusagebasedcostbasis_id" to table: "charge_usage_based_cost_bases"
DROP INDEX "chargeusagebasedcostbasis_id";
-- reverse: create index "chargeusagebasedcostbasis_currency_id" to table: "charge_usage_based_cost_bases"
DROP INDEX "chargeusagebasedcostbasis_currency_id";
-- reverse: create index "chargeusagebasedcostbasis_currency_cost_basis_id" to table: "charge_usage_based_cost_bases"
DROP INDEX "chargeusagebasedcostbasis_currency_cost_basis_id";
-- reverse: create "charge_usage_based_cost_bases" table
DROP TABLE "charge_usage_based_cost_bases";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fee_cost_basis_charge_fk", DROP COLUMN "cost_basis_id";
-- reverse: create index "chargeflatfeecostbasis_resolved_cost_basis_id" to table: "charge_flat_fee_cost_bases"
DROP INDEX "chargeflatfeecostbasis_resolved_cost_basis_id";
-- reverse: create index "chargeflatfeecostbasis_namespace" to table: "charge_flat_fee_cost_bases"
DROP INDEX "chargeflatfeecostbasis_namespace";
-- reverse: create index "chargeflatfeecostbasis_id" to table: "charge_flat_fee_cost_bases"
DROP INDEX "chargeflatfeecostbasis_id";
-- reverse: create index "chargeflatfeecostbasis_currency_id" to table: "charge_flat_fee_cost_bases"
DROP INDEX "chargeflatfeecostbasis_currency_id";
-- reverse: create index "chargeflatfeecostbasis_currency_cost_basis_id" to table: "charge_flat_fee_cost_bases"
DROP INDEX "chargeflatfeecostbasis_currency_cost_basis_id";
-- reverse: create "charge_flat_fee_cost_bases" table
DROP TABLE "charge_flat_fee_cost_bases";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchase_cost_basis_charge_fk", DROP COLUMN "cost_basis_id";
-- reverse: create index "chargecreditpurchasecostbasis_resolved_cost_basis_id" to table: "charge_credit_purchase_cost_bases"
DROP INDEX "chargecreditpurchasecostbasis_resolved_cost_basis_id";
-- reverse: create index "chargecreditpurchasecostbasis_namespace" to table: "charge_credit_purchase_cost_bases"
DROP INDEX "chargecreditpurchasecostbasis_namespace";
-- reverse: create index "chargecreditpurchasecostbasis_id" to table: "charge_credit_purchase_cost_bases"
DROP INDEX "chargecreditpurchasecostbasis_id";
-- reverse: create index "chargecreditpurchasecostbasis_currency_id" to table: "charge_credit_purchase_cost_bases"
DROP INDEX "chargecreditpurchasecostbasis_currency_id";
-- reverse: create index "chargecreditpurchasecostbasis_currency_cost_basis_id" to table: "charge_credit_purchase_cost_bases"
DROP INDEX "chargecreditpurchasecostbasis_currency_cost_basis_id";
-- reverse: create "charge_credit_purchase_cost_bases" table
DROP TABLE "charge_credit_purchase_cost_bases";
