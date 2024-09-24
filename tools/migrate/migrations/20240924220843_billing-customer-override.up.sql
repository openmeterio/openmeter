-- create "billing_customer_overrides" table
CREATE TABLE "billing_customer_overrides" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "collection_alignment" character varying NULL,
  "item_collection_period_seconds" bigint NULL,
  "invoice_auto_advance" boolean NULL,
  "invoice_draft_period_seconds" bigint NULL,
  "invoice_due_after_seconds" bigint NULL,
  "invoice_collection_method" character varying NULL,
  "invoice_item_resolution" character varying NULL,
  "invoice_item_per_subject" boolean NULL,
  "billing_profile_id" character(26) NULL,
  "customer_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_customer_overrides_billing_profiles_billing_customer_ov" FOREIGN KEY ("billing_profile_id") REFERENCES "billing_profiles" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_customer_overrides_customers_billing_customer_override" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billing_customer_overrides_customer_id_key" to table: "billing_customer_overrides"
CREATE UNIQUE INDEX "billing_customer_overrides_customer_id_key" ON "billing_customer_overrides" ("customer_id");
-- create index "billingcustomeroverride_id" to table: "billing_customer_overrides"
CREATE UNIQUE INDEX "billingcustomeroverride_id" ON "billing_customer_overrides" ("id");
-- create index "billingcustomeroverride_namespace" to table: "billing_customer_overrides"
CREATE INDEX "billingcustomeroverride_namespace" ON "billing_customer_overrides" ("namespace");
-- create index "billingcustomeroverride_namespace_customer_id" to table: "billing_customer_overrides"
CREATE UNIQUE INDEX "billingcustomeroverride_namespace_customer_id" ON "billing_customer_overrides" ("namespace", "customer_id");
-- create index "billingcustomeroverride_namespace_id" to table: "billing_customer_overrides"
CREATE UNIQUE INDEX "billingcustomeroverride_namespace_id" ON "billing_customer_overrides" ("namespace", "id");
