-- create "billing_gathering_invoices" table
CREATE TABLE "billing_gathering_invoices" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "number" character varying NOT NULL,
  "currency" character varying(3) NOT NULL,
  "service_period_start" timestamptz NULL,
  "service_period_end" timestamptz NULL,
  "next_collection_at" timestamptz NULL,
  "schema_level" bigint NOT NULL DEFAULT 2,
  "customer_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_gathering_invoices_customers_billing_gathering_invoices" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "service_period_both_set_or_null" CHECK ((service_period_start IS NULL) = (service_period_end IS NULL)),
  CONSTRAINT "service_period_not_inverted" CHECK ((service_period_start IS NULL) OR (service_period_start <= service_period_end))
);
-- create index "billinggatheringinvoice_customer_id" to table: "billing_gathering_invoices"
CREATE INDEX "billinggatheringinvoice_customer_id" ON "billing_gathering_invoices" ("customer_id");
-- create index "billinggatheringinvoice_id" to table: "billing_gathering_invoices"
CREATE UNIQUE INDEX "billinggatheringinvoice_id" ON "billing_gathering_invoices" ("id");
-- create index "billinggatheringinvoice_namespace" to table: "billing_gathering_invoices"
CREATE INDEX "billinggatheringinvoice_namespace" ON "billing_gathering_invoices" ("namespace");
-- create index "billinggatheringinvoice_namespace_created_at" to table: "billing_gathering_invoices"
CREATE INDEX "billinggatheringinvoice_namespace_created_at" ON "billing_gathering_invoices" ("namespace", "created_at");
-- create index "billinggatheringinvoice_namespace_customer_id" to table: "billing_gathering_invoices"
CREATE INDEX "billinggatheringinvoice_namespace_customer_id" ON "billing_gathering_invoices" ("namespace", "customer_id");
-- create index "billinggatheringinvoice_namespace_customer_id_currency" to table: "billing_gathering_invoices"
CREATE UNIQUE INDEX "billinggatheringinvoice_namespace_customer_id_currency" ON "billing_gathering_invoices" ("namespace", "customer_id", "currency") WHERE (deleted_at IS NULL);
-- create index "billinggatheringinvoice_namespace_id" to table: "billing_gathering_invoices"
CREATE UNIQUE INDEX "billinggatheringinvoice_namespace_id" ON "billing_gathering_invoices" ("namespace", "id");
-- create index "billinggatheringinvoice_namespace_next_collection_at" to table: "billing_gathering_invoices"
CREATE INDEX "billinggatheringinvoice_namespace_next_collection_at" ON "billing_gathering_invoices" ("namespace", "next_collection_at");
-- create index "billinggatheringinvoice_namespace_updated_at" to table: "billing_gathering_invoices"
CREATE INDEX "billinggatheringinvoice_namespace_updated_at" ON "billing_gathering_invoices" ("namespace", "updated_at");
-- modify "billing_gathering_invoice_lines" table
ALTER TABLE "billing_gathering_invoice_lines" DROP CONSTRAINT "billing_gathering_line_invoice_fk", ADD CONSTRAINT "service_period_not_inverted" CHECK (service_period_start <= service_period_end), ADD CONSTRAINT "billing_gathering_line_invoice_fk" FOREIGN KEY ("invoice_id") REFERENCES "billing_gathering_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create index "billinggatheringinvoiceline_invoice_id" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_invoice_id" ON "billing_gathering_invoice_lines" ("invoice_id");
-- create "billing_invoice_search_v1s" view
CREATE VIEW "billing_invoice_search_v1s" AS
SELECT "billing_invoices"."id", "billing_invoices"."namespace", "billing_invoices"."customer_id", "billing_invoices"."customer_name", "billing_invoices"."currency", 'billing_invoice' AS "storage_table", "billing_invoices"."type" AS "invoice_type", "billing_invoices"."status", "billing_invoices"."issued_at", "billing_invoices"."period_start" AS "service_period_start", "billing_invoices"."period_end" AS "service_period_end", "billing_invoices"."created_at", "billing_invoices"."updated_at", "billing_invoices"."deleted_at", "billing_invoices"."draft_until", "billing_invoices"."collection_at", "billing_invoices"."status_details_cache", "billing_invoices"."invoicing_app_external_id", "billing_invoices"."payment_app_external_id", "billing_invoices"."tax_app_external_id", "billing_invoices"."schema_level" FROM "billing_invoices" WHERE "billing_invoices"."status" <> 'gathering' UNION ALL SELECT "billing_invoices"."id", "billing_invoices"."namespace", "billing_invoices"."customer_id", "t1"."name" AS "customer_name", "billing_invoices"."currency", 'billing_invoice' AS "storage_table", 'gathering' AS "invoice_type", 'gathering' AS "status", NULL::timestamptz AS "issued_at", "billing_invoices"."period_start" AS "service_period_start", "billing_invoices"."period_end" AS "service_period_end", "billing_invoices"."created_at", "billing_invoices"."updated_at", "billing_invoices"."deleted_at", NULL::timestamptz AS "draft_until", "billing_invoices"."collection_at", NULL::jsonb AS "status_details_cache", NULL::text AS "invoicing_app_external_id", NULL::text AS "payment_app_external_id", NULL::text AS "tax_app_external_id", "billing_invoices"."schema_level" FROM "billing_invoices" JOIN "customers" AS "t1" ON "t1"."namespace" = "billing_invoices"."namespace" AND "t1"."id" = "billing_invoices"."customer_id" LEFT JOIN "billing_gathering_invoices" AS "t2" ON "t2"."namespace" = "billing_invoices"."namespace" AND "t2"."id" = "billing_invoices"."id" WHERE "billing_invoices"."status" = 'gathering' AND "t2"."id" IS NULL UNION ALL SELECT "billing_gathering_invoices"."id", "billing_gathering_invoices"."namespace", "billing_gathering_invoices"."customer_id", "t1"."name" AS "customer_name", "billing_gathering_invoices"."currency", 'billing_gathering_invoice' AS "storage_table", 'gathering' AS "invoice_type", 'gathering' AS "status", NULL::timestamptz AS "issued_at", "billing_gathering_invoices"."service_period_start", "billing_gathering_invoices"."service_period_end", "billing_gathering_invoices"."created_at", "billing_gathering_invoices"."updated_at", "billing_gathering_invoices"."deleted_at", NULL::timestamptz AS "draft_until", "billing_gathering_invoices"."next_collection_at" AS "collection_at", NULL::jsonb AS "status_details_cache", NULL::text AS "invoicing_app_external_id", NULL::text AS "payment_app_external_id", NULL::text AS "tax_app_external_id", "billing_gathering_invoices"."schema_level" FROM "billing_gathering_invoices" JOIN "customers" AS "t1" ON "t1"."namespace" = "billing_gathering_invoices"."namespace" AND "t1"."id" = "billing_gathering_invoices"."customer_id";
