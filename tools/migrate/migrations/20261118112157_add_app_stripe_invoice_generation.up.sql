-- create "app_stripe_invoice_generations" table
CREATE TABLE "app_stripe_invoice_generations" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "invoice_id" character(26) NOT NULL,
  "generation" bigint NOT NULL DEFAULT 0,
  PRIMARY KEY ("id")
);
-- create index "appstripeinvoicegeneration_id" to table: "app_stripe_invoice_generations"
CREATE UNIQUE INDEX "appstripeinvoicegeneration_id" ON "app_stripe_invoice_generations" ("id");
-- create index "appstripeinvoicegeneration_namespace" to table: "app_stripe_invoice_generations"
CREATE INDEX "appstripeinvoicegeneration_namespace" ON "app_stripe_invoice_generations" ("namespace");
-- create index "appstripeinvoicegeneration_namespace_invoice_id" to table: "app_stripe_invoice_generations"
CREATE UNIQUE INDEX "appstripeinvoicegeneration_namespace_invoice_id" ON "app_stripe_invoice_generations" ("namespace", "invoice_id");
