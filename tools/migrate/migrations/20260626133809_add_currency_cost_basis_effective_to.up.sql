-- modify "currency_cost_bases" table
ALTER TABLE "currency_cost_bases" DROP CONSTRAINT "currency_cost_bases_custom_currencies_cost_basis_history";
-- rename a column from "custom_currency_id" to "currency_id"
-- atlas:nolint BC102
ALTER TABLE "currency_cost_bases" RENAME COLUMN "custom_currency_id" TO "currency_id";
ALTER INDEX "currencycostbasis_namespace_custom_currency_id_fiat_code_effect" RENAME TO "currencycostbasis_namespace_currency_id_fiat_code_effective_fro";
ALTER TABLE "currency_cost_bases" ADD COLUMN "effective_to" timestamptz NULL;
ALTER TABLE "currency_cost_bases" ADD CONSTRAINT "currency_cost_bases_custom_currencies_cost_basis_history" FOREIGN KEY ("currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
