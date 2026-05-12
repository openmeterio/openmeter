-- reverse: drop "charge_flat_fee_payments" table
CREATE TABLE "charge_flat_fee_payments" (
  "id" character(26) NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "line_id" character(26) NOT NULL,
  "charge_id" character(26) NOT NULL,
  "invoice_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "charge_flat_fee_payments_charge_id_key" ON "charge_flat_fee_payments" ("charge_id");
CREATE UNIQUE INDEX "charge_flat_fee_payments_line_id_key" ON "charge_flat_fee_payments" ("line_id");
CREATE INDEX "chargeflatfeepayment_annotations" ON "charge_flat_fee_payments" USING gin ("annotations");
CREATE UNIQUE INDEX "chargeflatfeepayment_id" ON "charge_flat_fee_payments" ("id");
CREATE INDEX "chargeflatfeepayment_namespace" ON "charge_flat_fee_payments" ("namespace");
-- reverse: drop "charge_flat_fee_invoiced_usages" table
CREATE TABLE "charge_flat_fee_invoiced_usages" (
  "id" character(26) NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "mutable" boolean NOT NULL,
  "ledger_transaction_group_id" character(26) NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "line_id" character(26) NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "charge_flat_fee_invoiced_usages_charge_id_key" ON "charge_flat_fee_invoiced_usages" ("charge_id");
CREATE INDEX "chargeflatfeeinvoicedusage_annotations" ON "charge_flat_fee_invoiced_usages" USING gin ("annotations");
CREATE UNIQUE INDEX "chargeflatfeeinvoicedusage_id" ON "charge_flat_fee_invoiced_usages" ("id");
CREATE INDEX "chargeflatfeeinvoicedusage_namespace" ON "charge_flat_fee_invoiced_usages" ("namespace");
CREATE UNIQUE INDEX "chargeflatfeeinvoicedusage_namespace_charge_id" ON "charge_flat_fee_invoiced_usages" ("namespace", "charge_id");
-- reverse: drop "charge_flat_fee_detailed_line" table
CREATE TABLE "charge_flat_fee_detailed_line" (
  "id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "tax_config" jsonb NULL,
  "tax_behavior" character varying NULL,
  "service_period_start" timestamptz NOT NULL,
  "service_period_end" timestamptz NOT NULL,
  "quantity" numeric NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "child_unique_reference_id" character varying NOT NULL,
  "per_unit_amount" numeric NOT NULL,
  "category" character varying NOT NULL DEFAULT 'regular',
  "payment_term" character varying NOT NULL DEFAULT 'in_advance',
  "index" bigint NULL,
  "credits_applied" jsonb NULL,
  "annotations" jsonb NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "charge_id" character(26) NOT NULL,
  "tax_code_id" character(26) NULL,
  "pricer_reference_id" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "child_unique_reference_id_not_empty" CHECK ((child_unique_reference_id)::text <> ''::text)
);
CREATE UNIQUE INDEX "chargeffdetailedline_ns_charge_child_id" ON "charge_flat_fee_detailed_line" ("namespace", "charge_id", "child_unique_reference_id") WHERE (deleted_at IS NULL);
CREATE INDEX "chargeflatfeedetailedline_annotations" ON "charge_flat_fee_detailed_line" USING gin ("annotations");
CREATE UNIQUE INDEX "chargeflatfeedetailedline_id" ON "charge_flat_fee_detailed_line" ("id");
CREATE INDEX "chargeflatfeedetailedline_namespace" ON "charge_flat_fee_detailed_line" ("namespace");
CREATE INDEX "chargeflatfeedetailedline_namespace_charge_id" ON "charge_flat_fee_detailed_line" ("namespace", "charge_id");
CREATE UNIQUE INDEX "chargeflatfeedetailedline_namespace_id" ON "charge_flat_fee_detailed_line" ("namespace", "id");
CREATE INDEX "chargeflatfeedetailedline_tax_code_id" ON "charge_flat_fee_detailed_line" ("tax_code_id");
-- reverse: drop "charge_flat_fee_credit_allocations" table
CREATE TABLE "charge_flat_fee_credit_allocations" (
  "id" character(26) NOT NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "line_id" character(26) NULL,
  "charge_id" character(26) NOT NULL,
  "sort_hint" bigint NOT NULL,
  "type" character varying NOT NULL,
  "corrects_realization_id" character(26) NULL,
  PRIMARY KEY ("id")
);
CREATE INDEX "chargeflatfeecreditallocations_annotations" ON "charge_flat_fee_credit_allocations" USING gin ("annotations");
CREATE UNIQUE INDEX "chargeflatfeecreditallocations_id" ON "charge_flat_fee_credit_allocations" ("id");
CREATE INDEX "chargeflatfeecreditallocations_namespace" ON "charge_flat_fee_credit_allocations" ("namespace");
-- copy run-owned payment state back to the previous flat-fee payment table
INSERT INTO "charge_flat_fee_payments" (
  "id",
  "service_period_from",
  "service_period_to",
  "status",
  "amount",
  "authorized_transaction_group_id",
  "authorized_at",
  "settled_transaction_group_id",
  "settled_at",
  "namespace",
  "created_at",
  "updated_at",
  "deleted_at",
  "annotations",
  "line_id",
  "charge_id",
  "invoice_id"
)
SELECT
  "p"."id",
  "p"."service_period_from",
  "p"."service_period_to",
  "p"."status",
  "p"."amount",
  "p"."authorized_transaction_group_id",
  "p"."authorized_at",
  "p"."settled_transaction_group_id",
  "p"."settled_at",
  "p"."namespace",
  "p"."created_at",
  "p"."updated_at",
  "p"."deleted_at",
  "p"."annotations",
  "p"."line_id",
  "r"."charge_id",
  "p"."invoice_id"
FROM "charge_flat_fee_run_payments" AS "p"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."id" = "p"."run_id"
JOIN "charge_flat_fees" AS "ff"
  ON "ff"."namespace" = "r"."namespace"
  AND "ff"."id" = "r"."charge_id"
  AND "ff"."current_realization_run_id" = "p"."run_id";
-- copy run-owned invoiced usage back to the previous flat-fee invoiced usage table
INSERT INTO "charge_flat_fee_invoiced_usages" (
  "id",
  "service_period_from",
  "service_period_to",
  "mutable",
  "ledger_transaction_group_id",
  "namespace",
  "created_at",
  "updated_at",
  "deleted_at",
  "annotations",
  "amount",
  "taxes_total",
  "taxes_inclusive_total",
  "taxes_exclusive_total",
  "charges_total",
  "discounts_total",
  "credits_total",
  "total",
  "line_id",
  "charge_id"
)
SELECT
  "iu"."id",
  "iu"."service_period_from",
  "iu"."service_period_to",
  "iu"."mutable",
  "iu"."ledger_transaction_group_id",
  "iu"."namespace",
  "iu"."created_at",
  "iu"."updated_at",
  "iu"."deleted_at",
  "iu"."annotations",
  "iu"."amount",
  "iu"."taxes_total",
  "iu"."taxes_inclusive_total",
  "iu"."taxes_exclusive_total",
  "iu"."charges_total",
  "iu"."discounts_total",
  "iu"."credits_total",
  "iu"."total",
  "iu"."line_id",
  "r"."charge_id"
FROM "charge_flat_fee_run_invoiced_usages" AS "iu"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."id" = "iu"."run_id"
JOIN "charge_flat_fees" AS "ff"
  ON "ff"."namespace" = "r"."namespace"
  AND "ff"."id" = "r"."charge_id"
  AND "ff"."current_realization_run_id" = "iu"."run_id";
-- copy run-owned detailed lines back to the previous flat-fee detailed line table
INSERT INTO "charge_flat_fee_detailed_line" (
  "id",
  "currency",
  "tax_config",
  "tax_behavior",
  "service_period_start",
  "service_period_end",
  "quantity",
  "invoicing_app_external_id",
  "child_unique_reference_id",
  "per_unit_amount",
  "category",
  "payment_term",
  "index",
  "credits_applied",
  "annotations",
  "namespace",
  "metadata",
  "created_at",
  "updated_at",
  "deleted_at",
  "name",
  "description",
  "amount",
  "taxes_total",
  "taxes_inclusive_total",
  "taxes_exclusive_total",
  "charges_total",
  "discounts_total",
  "credits_total",
  "total",
  "charge_id",
  "tax_code_id",
  "pricer_reference_id"
)
SELECT
  "dl"."id",
  "dl"."currency",
  "dl"."tax_config",
  "dl"."tax_behavior",
  "dl"."service_period_start",
  "dl"."service_period_end",
  "dl"."quantity",
  "dl"."invoicing_app_external_id",
  "dl"."child_unique_reference_id",
  "dl"."per_unit_amount",
  "dl"."category",
  "dl"."payment_term",
  "dl"."index",
  "dl"."credits_applied",
  "dl"."annotations",
  "dl"."namespace",
  "dl"."metadata",
  "dl"."created_at",
  "dl"."updated_at",
  "dl"."deleted_at",
  "dl"."name",
  "dl"."description",
  "dl"."amount",
  "dl"."taxes_total",
  "dl"."taxes_inclusive_total",
  "dl"."taxes_exclusive_total",
  "dl"."charges_total",
  "dl"."discounts_total",
  "dl"."credits_total",
  "dl"."total",
  "r"."charge_id",
  "dl"."tax_code_id",
  "dl"."pricer_reference_id"
FROM "charge_flat_fee_run_detailed_lines" AS "dl"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."id" = "dl"."run_id"
JOIN "charge_flat_fees" AS "ff"
  ON "ff"."namespace" = "r"."namespace"
  AND "ff"."id" = "r"."charge_id"
  AND "ff"."current_realization_run_id" = "dl"."run_id";
-- copy run-owned credit allocations back to the previous flat-fee credit allocation table
INSERT INTO "charge_flat_fee_credit_allocations" (
  "id",
  "amount",
  "service_period_from",
  "service_period_to",
  "ledger_transaction_group_id",
  "namespace",
  "created_at",
  "updated_at",
  "deleted_at",
  "annotations",
  "line_id",
  "charge_id",
  "sort_hint",
  "type",
  "corrects_realization_id"
)
SELECT
  "ca"."id",
  "ca"."amount",
  "ca"."service_period_from",
  "ca"."service_period_to",
  "ca"."ledger_transaction_group_id",
  "ca"."namespace",
  "ca"."created_at",
  "ca"."updated_at",
  "ca"."deleted_at",
  "ca"."annotations",
  "ca"."line_id",
  "r"."charge_id",
  "ca"."sort_hint",
  "ca"."type",
  "ca"."corrects_realization_id"
FROM "charge_flat_fee_run_credit_allocations" AS "ca"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."id" = "ca"."run_id";
-- reverse: modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" DROP CONSTRAINT "charge_flat_fee_runs_charge_flat_fees_runs";
-- reverse: modify "charge_flat_fee_run_payments" table
ALTER TABLE "charge_flat_fee_run_payments" DROP CONSTRAINT "charge_flat_fee_run_payments_charge_flat_fee_runs_payment", DROP CONSTRAINT "charge_flat_fee_run_payments_billing_invoice_lines_charge_flat_";
-- reverse: modify "charge_flat_fee_run_invoiced_usages" table
ALTER TABLE "charge_flat_fee_run_invoiced_usages" DROP CONSTRAINT "charge_flat_fee_run_invoiced_usages_charge_flat_fee_runs_invoic", DROP CONSTRAINT "charge_flat_fee_run_invoiced_usages_billing_invoice_lines_charg";
-- reverse: modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" DROP CONSTRAINT "charge_flat_fee_run_detailed_lines_tax_codes_charge_flat_fee_ru", DROP CONSTRAINT "charge_flat_fee_run_detailed_lines_charge_flat_fee_runs_detaile";
-- reverse: modify "charge_flat_fee_run_credit_allocations" table
ALTER TABLE "charge_flat_fee_run_credit_allocations" DROP CONSTRAINT "charge_flat_fee_run_credit_allocations_billing_invoice_lines_ch", DROP CONSTRAINT "charge_ff_credit_alloc_run";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_charge_flat_fee_runs_current_run";
-- reverse: create index "chargeflatfeerun_namespace_charge_id" to table: "charge_flat_fee_runs"
DROP INDEX "chargeflatfeerun_namespace_charge_id";
-- reverse: create index "chargeflatfeerun_namespace" to table: "charge_flat_fee_runs"
DROP INDEX "chargeflatfeerun_namespace";
-- reverse: create index "chargeflatfeerun_id" to table: "charge_flat_fee_runs"
DROP INDEX "chargeflatfeerun_id";
-- reverse: create "charge_flat_fee_runs" table
DROP TABLE "charge_flat_fee_runs";
-- reverse: create index "chargeflatfeerunpayment_namespace" to table: "charge_flat_fee_run_payments"
DROP INDEX "chargeflatfeerunpayment_namespace";
-- reverse: create index "chargeflatfeerunpayment_id" to table: "charge_flat_fee_run_payments"
DROP INDEX "chargeflatfeerunpayment_id";
-- reverse: create index "chargeflatfeerunpayment_annotations" to table: "charge_flat_fee_run_payments"
DROP INDEX "chargeflatfeerunpayment_annotations";
-- reverse: create index "charge_flat_fee_run_payments_run_id_key" to table: "charge_flat_fee_run_payments"
DROP INDEX "charge_flat_fee_run_payments_run_id_key";
-- reverse: create index "charge_flat_fee_run_payments_line_id_key" to table: "charge_flat_fee_run_payments"
DROP INDEX "charge_flat_fee_run_payments_line_id_key";
-- reverse: create "charge_flat_fee_run_payments" table
DROP TABLE "charge_flat_fee_run_payments";
-- reverse: create index "chargeflatfeeruninvoicedusage_namespace_run_id" to table: "charge_flat_fee_run_invoiced_usages"
DROP INDEX "chargeflatfeeruninvoicedusage_namespace_run_id";
-- reverse: create index "chargeflatfeeruninvoicedusage_namespace" to table: "charge_flat_fee_run_invoiced_usages"
DROP INDEX "chargeflatfeeruninvoicedusage_namespace";
-- reverse: create index "chargeflatfeeruninvoicedusage_id" to table: "charge_flat_fee_run_invoiced_usages"
DROP INDEX "chargeflatfeeruninvoicedusage_id";
-- reverse: create index "chargeflatfeeruninvoicedusage_annotations" to table: "charge_flat_fee_run_invoiced_usages"
DROP INDEX "chargeflatfeeruninvoicedusage_annotations";
-- reverse: create index "charge_flat_fee_run_invoiced_usages_run_id_key" to table: "charge_flat_fee_run_invoiced_usages"
DROP INDEX "charge_flat_fee_run_invoiced_usages_run_id_key";
-- reverse: create "charge_flat_fee_run_invoiced_usages" table
DROP TABLE "charge_flat_fee_run_invoiced_usages";
-- reverse: create index "chargeflatfeerundetailedline_tax_code_id" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeflatfeerundetailedline_tax_code_id";
-- reverse: create index "chargeflatfeerundetailedline_namespace_run_id" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeflatfeerundetailedline_namespace_run_id";
-- reverse: create index "chargeflatfeerundetailedline_namespace_id" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeflatfeerundetailedline_namespace_id";
-- reverse: create index "chargeflatfeerundetailedline_namespace" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeflatfeerundetailedline_namespace";
-- reverse: create index "chargeflatfeerundetailedline_id" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeflatfeerundetailedline_id";
-- reverse: create index "chargeflatfeerundetailedline_annotations" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeflatfeerundetailedline_annotations";
-- reverse: create index "chargeffdetailedline_ns_run_child_id" to table: "charge_flat_fee_run_detailed_lines"
DROP INDEX "chargeffdetailedline_ns_run_child_id";
-- reverse: create "charge_flat_fee_run_detailed_lines" table
DROP TABLE "charge_flat_fee_run_detailed_lines";
-- reverse: create index "chargeflatfeeruncreditallocations_namespace" to table: "charge_flat_fee_run_credit_allocations"
DROP INDEX "chargeflatfeeruncreditallocations_namespace";
-- reverse: create index "chargeflatfeeruncreditallocations_id" to table: "charge_flat_fee_run_credit_allocations"
DROP INDEX "chargeflatfeeruncreditallocations_id";
-- reverse: create index "chargeflatfeeruncreditallocations_annotations" to table: "charge_flat_fee_run_credit_allocations"
DROP INDEX "chargeflatfeeruncreditallocations_annotations";
-- reverse: create "charge_flat_fee_run_credit_allocations" table
DROP TABLE "charge_flat_fee_run_credit_allocations";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP COLUMN "current_realization_run_id";
-- reverse: modify "charge_flat_fee_payments" table
ALTER TABLE "charge_flat_fee_payments" ADD CONSTRAINT "charge_flat_fee_payments_charge_flat_fees_payment" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charge_flat_fee_payments_billing_invoice_lines_charge_flat_fee_" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "charge_flat_fee_invoiced_usages" table
ALTER TABLE "charge_flat_fee_invoiced_usages" ADD CONSTRAINT "charge_flat_fee_invoiced_usages_charge_flat_fees_invoiced_usage" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charge_flat_fee_invoiced_usages_billing_invoice_lines_charge_fl" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: modify "charge_flat_fee_detailed_line" table
ALTER TABLE "charge_flat_fee_detailed_line" ADD CONSTRAINT "charge_flat_fee_detailed_line_tax_codes_charge_flat_fee_detaile" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_flat_fee_detailed_line_charge_flat_fees_detailed_lines" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: modify "charge_flat_fee_credit_allocations" table
ALTER TABLE "charge_flat_fee_credit_allocations" ADD CONSTRAINT "charge_flat_fee_credit_allocations_billing_invoice_lines_charge" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_ff_credit_alloc_flat_fee" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charge_flat_fee_credit_allocations_charge_flat_fee_credit_alloc" FOREIGN KEY ("corrects_realization_id") REFERENCES "charge_flat_fee_credit_allocations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
