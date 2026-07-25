-- reverse: modify "currency_cost_bases" table
ALTER TABLE "currency_cost_bases" DROP CONSTRAINT "currency_cost_bases_custom_currencies_cost_basis_history";
ALTER TABLE "currency_cost_bases" DROP COLUMN "effective_to";
ALTER INDEX "currencycostbasis_namespace_currency_id_fiat_code_effective_fro" RENAME TO "currencycostbasis_namespace_custom_currency_id_fiat_code_effect";
ALTER TABLE "currency_cost_bases" RENAME COLUMN "currency_id" TO "custom_currency_id";
ALTER TABLE "currency_cost_bases" ADD CONSTRAINT "currency_cost_bases_custom_currencies_cost_basis_history" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
