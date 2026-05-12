-- modify "charge_flat_fee_credit_allocations" table
ALTER TABLE "charge_flat_fee_credit_allocations" DROP CONSTRAINT "charge_ff_credit_alloc_flat_fee", DROP CONSTRAINT "charge_flat_fee_credit_allocations_billing_invoice_lines_charge";
-- modify "charge_flat_fee_detailed_line" table
ALTER TABLE "charge_flat_fee_detailed_line" DROP CONSTRAINT "charge_flat_fee_detailed_line_charge_flat_fees_detailed_lines", DROP CONSTRAINT "charge_flat_fee_detailed_line_tax_codes_charge_flat_fee_detaile";
-- modify "charge_flat_fee_invoiced_usages" table
ALTER TABLE "charge_flat_fee_invoiced_usages" DROP CONSTRAINT "charge_flat_fee_invoiced_usages_billing_invoice_lines_charge_fl", DROP CONSTRAINT "charge_flat_fee_invoiced_usages_charge_flat_fees_invoiced_usage";
-- modify "charge_flat_fee_payments" table
ALTER TABLE "charge_flat_fee_payments" DROP CONSTRAINT "charge_flat_fee_payments_billing_invoice_lines_charge_flat_fee_", DROP CONSTRAINT "charge_flat_fee_payments_charge_flat_fees_payment";
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "current_realization_run_id" character(26) NULL;
-- create "charge_flat_fee_run_credit_allocations" table
CREATE TABLE "charge_flat_fee_run_credit_allocations" (
  "id" character(26) NOT NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "sort_hint" bigint NOT NULL,
  "type" character varying NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "line_id" character(26) NULL,
  "run_id" character(26) NOT NULL,
  "corrects_realization_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_run_credit_allocations_charge_flat_fee_run_cred" FOREIGN KEY ("corrects_realization_id") REFERENCES "charge_flat_fee_run_credit_allocations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "chargeflatfeeruncreditallocations_annotations" to table: "charge_flat_fee_run_credit_allocations"
CREATE INDEX "chargeflatfeeruncreditallocations_annotations" ON "charge_flat_fee_run_credit_allocations" USING gin ("annotations");
-- create index "chargeflatfeeruncreditallocations_id" to table: "charge_flat_fee_run_credit_allocations"
CREATE UNIQUE INDEX "chargeflatfeeruncreditallocations_id" ON "charge_flat_fee_run_credit_allocations" ("id");
-- create index "chargeflatfeeruncreditallocations_namespace" to table: "charge_flat_fee_run_credit_allocations"
CREATE INDEX "chargeflatfeeruncreditallocations_namespace" ON "charge_flat_fee_run_credit_allocations" ("namespace");
-- create "charge_flat_fee_run_detailed_lines" table
CREATE TABLE "charge_flat_fee_run_detailed_lines" (
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
  "pricer_reference_id" character varying NOT NULL,
  "run_id" character(26) NOT NULL,
  "tax_code_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "child_unique_reference_id_not_empty" CHECK ((child_unique_reference_id)::text <> ''::text)
);
-- create index "chargeffdetailedline_ns_run_child_id" to table: "charge_flat_fee_run_detailed_lines"
CREATE UNIQUE INDEX "chargeffdetailedline_ns_run_child_id" ON "charge_flat_fee_run_detailed_lines" ("namespace", "run_id", "child_unique_reference_id") WHERE (deleted_at IS NULL);
-- create index "chargeflatfeerundetailedline_annotations" to table: "charge_flat_fee_run_detailed_lines"
CREATE INDEX "chargeflatfeerundetailedline_annotations" ON "charge_flat_fee_run_detailed_lines" USING gin ("annotations");
-- create index "chargeflatfeerundetailedline_id" to table: "charge_flat_fee_run_detailed_lines"
CREATE UNIQUE INDEX "chargeflatfeerundetailedline_id" ON "charge_flat_fee_run_detailed_lines" ("id");
-- create index "chargeflatfeerundetailedline_namespace" to table: "charge_flat_fee_run_detailed_lines"
CREATE INDEX "chargeflatfeerundetailedline_namespace" ON "charge_flat_fee_run_detailed_lines" ("namespace");
-- create index "chargeflatfeerundetailedline_namespace_id" to table: "charge_flat_fee_run_detailed_lines"
CREATE UNIQUE INDEX "chargeflatfeerundetailedline_namespace_id" ON "charge_flat_fee_run_detailed_lines" ("namespace", "id");
-- create index "chargeflatfeerundetailedline_namespace_run_id" to table: "charge_flat_fee_run_detailed_lines"
CREATE INDEX "chargeflatfeerundetailedline_namespace_run_id" ON "charge_flat_fee_run_detailed_lines" ("namespace", "run_id");
-- create index "chargeflatfeerundetailedline_tax_code_id" to table: "charge_flat_fee_run_detailed_lines"
CREATE INDEX "chargeflatfeerundetailedline_tax_code_id" ON "charge_flat_fee_run_detailed_lines" ("tax_code_id");
-- create "charge_flat_fee_run_invoiced_usages" table
CREATE TABLE "charge_flat_fee_run_invoiced_usages" (
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
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "charge_flat_fee_run_invoiced_usages_run_id_key" to table: "charge_flat_fee_run_invoiced_usages"
CREATE UNIQUE INDEX "charge_flat_fee_run_invoiced_usages_run_id_key" ON "charge_flat_fee_run_invoiced_usages" ("run_id");
-- create index "chargeflatfeeruninvoicedusage_annotations" to table: "charge_flat_fee_run_invoiced_usages"
CREATE INDEX "chargeflatfeeruninvoicedusage_annotations" ON "charge_flat_fee_run_invoiced_usages" USING gin ("annotations");
-- create index "chargeflatfeeruninvoicedusage_id" to table: "charge_flat_fee_run_invoiced_usages"
CREATE UNIQUE INDEX "chargeflatfeeruninvoicedusage_id" ON "charge_flat_fee_run_invoiced_usages" ("id");
-- create index "chargeflatfeeruninvoicedusage_namespace" to table: "charge_flat_fee_run_invoiced_usages"
CREATE INDEX "chargeflatfeeruninvoicedusage_namespace" ON "charge_flat_fee_run_invoiced_usages" ("namespace");
-- create index "chargeflatfeeruninvoicedusage_namespace_run_id" to table: "charge_flat_fee_run_invoiced_usages"
CREATE UNIQUE INDEX "chargeflatfeeruninvoicedusage_namespace_run_id" ON "charge_flat_fee_run_invoiced_usages" ("namespace", "run_id");
-- create "charge_flat_fee_run_payments" table
CREATE TABLE "charge_flat_fee_run_payments" (
  "id" character(26) NOT NULL,
  "invoice_id" character(26) NOT NULL,
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
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "charge_flat_fee_run_payments_line_id_key" to table: "charge_flat_fee_run_payments"
CREATE UNIQUE INDEX "charge_flat_fee_run_payments_line_id_key" ON "charge_flat_fee_run_payments" ("line_id");
-- create index "charge_flat_fee_run_payments_run_id_key" to table: "charge_flat_fee_run_payments"
CREATE UNIQUE INDEX "charge_flat_fee_run_payments_run_id_key" ON "charge_flat_fee_run_payments" ("run_id");
-- create index "chargeflatfeerunpayment_annotations" to table: "charge_flat_fee_run_payments"
CREATE INDEX "chargeflatfeerunpayment_annotations" ON "charge_flat_fee_run_payments" USING gin ("annotations");
-- create index "chargeflatfeerunpayment_id" to table: "charge_flat_fee_run_payments"
CREATE UNIQUE INDEX "chargeflatfeerunpayment_id" ON "charge_flat_fee_run_payments" ("id");
-- create index "chargeflatfeerunpayment_namespace" to table: "charge_flat_fee_run_payments"
CREATE INDEX "chargeflatfeerunpayment_namespace" ON "charge_flat_fee_run_payments" ("namespace");
-- create "charge_flat_fee_runs" table
CREATE TABLE "charge_flat_fee_runs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "type" character varying NOT NULL,
  "initial_type" character varying NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "amount_after_proration" numeric NOT NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "chargeflatfeerun_id" to table: "charge_flat_fee_runs"
CREATE UNIQUE INDEX "chargeflatfeerun_id" ON "charge_flat_fee_runs" ("id");
-- create index "chargeflatfeerun_namespace" to table: "charge_flat_fee_runs"
CREATE INDEX "chargeflatfeerun_namespace" ON "charge_flat_fee_runs" ("namespace");
-- create index "chargeflatfeerun_namespace_charge_id" to table: "charge_flat_fee_runs"
CREATE INDEX "chargeflatfeerun_namespace_charge_id" ON "charge_flat_fee_runs" ("namespace", "charge_id");
-- backfill one realization run for each existing flat fee charge
INSERT INTO "charge_flat_fee_runs" (
  "id",
  "namespace",
  "created_at",
  "updated_at",
  "deleted_at",
  "amount",
  "taxes_total",
  "taxes_inclusive_total",
  "taxes_exclusive_total",
  "charges_total",
  "discounts_total",
  "credits_total",
  "total",
  "type",
  "initial_type",
  "service_period_from",
  "service_period_to",
  "amount_after_proration",
  "charge_id"
)
SELECT
  om_func_generate_ulid(),
  "ff"."namespace",
  "ff"."created_at",
  "ff"."updated_at",
  "ff"."deleted_at",
  COALESCE(
    "iu"."amount",
    CASE WHEN "ff"."settlement_mode" = 'credit_only' THEN COALESCE("ca"."amount", 0) ELSE 0 END
  ),
  COALESCE("iu"."taxes_total", 0),
  COALESCE("iu"."taxes_inclusive_total", 0),
  COALESCE("iu"."taxes_exclusive_total", 0),
  COALESCE(
    "iu"."charges_total",
    CASE WHEN "ff"."settlement_mode" = 'credit_only' THEN COALESCE("ca"."amount", 0) ELSE 0 END
  ),
  COALESCE("iu"."discounts_total", 0),
  COALESCE(
    "iu"."credits_total",
    CASE WHEN "ff"."settlement_mode" = 'credit_only' THEN COALESCE("ca"."amount", 0) ELSE 0 END
  ),
  COALESCE("iu"."total", 0),
  'final_realization',
  'final_realization',
  "ff"."service_period_from",
  "ff"."service_period_to",
  "ff"."amount_after_proration",
  "ff"."id"
FROM "charge_flat_fees" AS "ff"
LEFT JOIN "charge_flat_fee_invoiced_usages" AS "iu"
  ON "iu"."namespace" = "ff"."namespace"
  AND "iu"."charge_id" = "ff"."id"
LEFT JOIN (
  SELECT
    "namespace",
    "charge_id",
    GREATEST(COALESCE(SUM("amount"), 0), 0) AS "amount"
  FROM "charge_flat_fee_credit_allocations"
  WHERE "deleted_at" IS NULL
  GROUP BY "namespace", "charge_id"
) AS "ca"
  ON "ca"."namespace" = "ff"."namespace"
  AND "ca"."charge_id" = "ff"."id";
-- mark the backfilled run as current on the parent flat fee
UPDATE "charge_flat_fees" AS "ff"
SET "current_realization_run_id" = "r"."id"
FROM "charge_flat_fee_runs" AS "r"
WHERE "r"."namespace" = "ff"."namespace"
  AND "r"."charge_id" = "ff"."id";
-- copy existing credit allocations to the run-owned table while preserving IDs for lineage
INSERT INTO "charge_flat_fee_run_credit_allocations" (
  "id",
  "amount",
  "service_period_from",
  "service_period_to",
  "ledger_transaction_group_id",
  "sort_hint",
  "type",
  "namespace",
  "created_at",
  "updated_at",
  "deleted_at",
  "annotations",
  "line_id",
  "run_id",
  "corrects_realization_id"
)
SELECT
  "ca"."id",
  "ca"."amount",
  "ca"."service_period_from",
  "ca"."service_period_to",
  "ca"."ledger_transaction_group_id",
  "ca"."sort_hint",
  "ca"."type",
  "ca"."namespace",
  "ca"."created_at",
  "ca"."updated_at",
  "ca"."deleted_at",
  "ca"."annotations",
  "ca"."line_id",
  "r"."id",
  NULL
FROM "charge_flat_fee_credit_allocations" AS "ca"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."namespace" = "ca"."namespace"
  AND "r"."charge_id" = "ca"."charge_id";
UPDATE "charge_flat_fee_run_credit_allocations" AS "new_ca"
SET "corrects_realization_id" = "old_ca"."corrects_realization_id"
FROM "charge_flat_fee_credit_allocations" AS "old_ca"
WHERE "new_ca"."namespace" = "old_ca"."namespace"
  AND "new_ca"."id" = "old_ca"."id"
  AND "old_ca"."corrects_realization_id" IS NOT NULL;
-- copy existing detailed lines to the run-owned table
INSERT INTO "charge_flat_fee_run_detailed_lines" (
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
  "pricer_reference_id",
  "run_id",
  "tax_code_id"
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
  "dl"."pricer_reference_id",
  "r"."id",
  "dl"."tax_code_id"
FROM "charge_flat_fee_detailed_line" AS "dl"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."namespace" = "dl"."namespace"
  AND "r"."charge_id" = "dl"."charge_id";
-- copy existing invoiced usage to the run-owned table
INSERT INTO "charge_flat_fee_run_invoiced_usages" (
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
  "run_id"
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
  "r"."id"
FROM "charge_flat_fee_invoiced_usages" AS "iu"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."namespace" = "iu"."namespace"
  AND "r"."charge_id" = "iu"."charge_id";
-- copy existing payment state to the run-owned table
INSERT INTO "charge_flat_fee_run_payments" (
  "id",
  "invoice_id",
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
  "run_id"
)
SELECT
  "p"."id",
  "p"."invoice_id",
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
  "r"."id"
FROM "charge_flat_fee_payments" AS "p"
JOIN "charge_flat_fee_runs" AS "r"
  ON "r"."namespace" = "p"."namespace"
  AND "r"."charge_id" = "p"."charge_id";
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD CONSTRAINT "charge_flat_fees_charge_flat_fee_runs_current_run" FOREIGN KEY ("current_realization_run_id") REFERENCES "charge_flat_fee_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_flat_fee_run_credit_allocations" table
ALTER TABLE "charge_flat_fee_run_credit_allocations" ADD CONSTRAINT "charge_ff_credit_alloc_run" FOREIGN KEY ("run_id") REFERENCES "charge_flat_fee_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charge_flat_fee_run_credit_allocations_billing_invoice_lines_ch" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" ADD CONSTRAINT "charge_flat_fee_run_detailed_lines_charge_flat_fee_runs_detaile" FOREIGN KEY ("run_id") REFERENCES "charge_flat_fee_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charge_flat_fee_run_detailed_lines_tax_codes_charge_flat_fee_ru" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_flat_fee_run_invoiced_usages" table
ALTER TABLE "charge_flat_fee_run_invoiced_usages" ADD CONSTRAINT "charge_flat_fee_run_invoiced_usages_billing_invoice_lines_charg" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_flat_fee_run_invoiced_usages_charge_flat_fee_runs_invoic" FOREIGN KEY ("run_id") REFERENCES "charge_flat_fee_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "charge_flat_fee_run_payments" table
ALTER TABLE "charge_flat_fee_run_payments" ADD CONSTRAINT "charge_flat_fee_run_payments_billing_invoice_lines_charge_flat_" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "charge_flat_fee_run_payments_charge_flat_fee_runs_payment" FOREIGN KEY ("run_id") REFERENCES "charge_flat_fee_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" ADD CONSTRAINT "charge_flat_fee_runs_charge_flat_fees_runs" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- drop "charge_flat_fee_credit_allocations" table
DROP TABLE "charge_flat_fee_credit_allocations";
-- drop "charge_flat_fee_detailed_line" table
DROP TABLE "charge_flat_fee_detailed_line";
-- drop "charge_flat_fee_invoiced_usages" table
DROP TABLE "charge_flat_fee_invoiced_usages";
-- drop "charge_flat_fee_payments" table
DROP TABLE "charge_flat_fee_payments";
