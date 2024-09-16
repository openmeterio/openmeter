-- create "billing_profiles" table
CREATE TABLE "billing_profiles" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "key" character varying NOT NULL,
  "provider_config" jsonb NOT NULL,
  "billing_config" jsonb NOT NULL,
  "default" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("id")
);
-- create index "billingprofile_id" to table: "billing_profiles"
CREATE INDEX "billingprofile_id" ON "billing_profiles" ("id");
-- create index "billingprofile_namespace_default" to table: "billing_profiles"
CREATE INDEX "billingprofile_namespace_default" ON "billing_profiles" ("namespace", "default");
-- create index "billingprofile_namespace_id" to table: "billing_profiles"
CREATE INDEX "billingprofile_namespace_id" ON "billing_profiles" ("namespace", "id");
-- create index "billingprofile_namespace_key" to table: "billing_profiles"
CREATE INDEX "billingprofile_namespace_key" ON "billing_profiles" ("namespace", "key");
-- create "billing_invoices" table
CREATE TABLE "billing_invoices" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "key" character varying NOT NULL,
  "customer_id" character(26) NOT NULL,
  "voided_at" timestamptz NULL,
  "currency" character varying(3) NOT NULL,
  "total_amount" numeric NOT NULL,
  "due_date" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "provider_config" jsonb NOT NULL,
  "billing_config" jsonb NOT NULL,
  "provider_reference" jsonb NOT NULL,
  "period_start" timestamptz NOT NULL,
  "period_end" timestamptz NOT NULL,
  "billing_profile_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoices_billing_profiles_billing_invoices" FOREIGN KEY ("billing_profile_id") REFERENCES "billing_profiles" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "billinginvoice_id" to table: "billing_invoices"
CREATE INDEX "billinginvoice_id" ON "billing_invoices" ("id");
-- create index "billinginvoice_namespace_customer_id" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_customer_id" ON "billing_invoices" ("namespace", "customer_id");
-- create index "billinginvoice_namespace_due_date" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_due_date" ON "billing_invoices" ("namespace", "due_date");
-- create index "billinginvoice_namespace_id" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_id" ON "billing_invoices" ("namespace", "id");
-- create index "billinginvoice_namespace_key" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_key" ON "billing_invoices" ("namespace", "key");
-- create index "billinginvoice_namespace_status" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_status" ON "billing_invoices" ("namespace", "status");
-- create "billing_invoice_items" table
CREATE TABLE "billing_invoice_items" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "customer_id" character(26) NOT NULL,
  "period_start" timestamptz NOT NULL,
  "period_end" timestamptz NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "quantity" numeric NOT NULL,
  "unit_price" numeric NOT NULL,
  "currency" character varying(3) NOT NULL,
  "tax_code_override" jsonb NOT NULL,
  "invoice_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_items_billing_invoices_billing_invoice_items" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "billinginvoiceitem_id" to table: "billing_invoice_items"
CREATE INDEX "billinginvoiceitem_id" ON "billing_invoice_items" ("id");
-- create index "billinginvoiceitem_namespace_customer_id" to table: "billing_invoice_items"
CREATE INDEX "billinginvoiceitem_namespace_customer_id" ON "billing_invoice_items" ("namespace", "customer_id");
-- create index "billinginvoiceitem_namespace_id" to table: "billing_invoice_items"
CREATE INDEX "billinginvoiceitem_namespace_id" ON "billing_invoice_items" ("namespace", "id");
-- create index "billinginvoiceitem_namespace_invoice_id" to table: "billing_invoice_items"
CREATE INDEX "billinginvoiceitem_namespace_invoice_id" ON "billing_invoice_items" ("namespace", "invoice_id");
