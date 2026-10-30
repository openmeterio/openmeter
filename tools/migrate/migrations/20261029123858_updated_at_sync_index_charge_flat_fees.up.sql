-- create index "chargeflatfee_namespace_updated_at" to table: "charge_flat_fees"
CREATE INDEX "chargeflatfee_namespace_updated_at" ON "charge_flat_fees" ("namespace", "updated_at");
