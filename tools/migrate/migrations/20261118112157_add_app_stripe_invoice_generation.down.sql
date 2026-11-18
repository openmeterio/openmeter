-- reverse: create index "appstripeinvoicegeneration_namespace_invoice_id" to table: "app_stripe_invoice_generations"
DROP INDEX "appstripeinvoicegeneration_namespace_invoice_id";
-- reverse: create index "appstripeinvoicegeneration_namespace" to table: "app_stripe_invoice_generations"
DROP INDEX "appstripeinvoicegeneration_namespace";
-- reverse: create index "appstripeinvoicegeneration_id" to table: "app_stripe_invoice_generations"
DROP INDEX "appstripeinvoicegeneration_id";
-- reverse: create "app_stripe_invoice_generations" table
DROP TABLE "app_stripe_invoice_generations";
