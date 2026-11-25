-- create "charge_usage_based_revenue_recognitions" table
CREATE TABLE "charge_usage_based_revenue_recognitions" (
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
  "stored_at_lt" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "metered_quantity" numeric NOT NULL,
  "detailed_lines_include_credit_allocations" boolean NOT NULL,
  "charge_id" character(26) NOT NULL,
  "prior_rev_recog_run_id" character(26) NULL,
  "feature_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_ub_revrec_prior_rev_recog_run" FOREIGN KEY ("prior_rev_recog_run_id") REFERENCES "charge_usage_based_revenue_recognitions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_usage_based_revenue_recognitions_charge_usage_based_reve" FOREIGN KEY ("charge_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "charge_usage_based_revenue_recognitions_features_usage_based_re" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "chargeusagebasedrevenuerecognitions_id" to table: "charge_usage_based_revenue_recognitions"
CREATE UNIQUE INDEX "chargeusagebasedrevenuerecognitions_id" ON "charge_usage_based_revenue_recognitions" ("id");
-- create index "chargeusagebasedrevenuerecognitions_namespace" to table: "charge_usage_based_revenue_recognitions"
CREATE INDEX "chargeusagebasedrevenuerecognitions_namespace" ON "charge_usage_based_revenue_recognitions" ("namespace");
-- create index "chargeusagebasedrevenuerecognitions_namespace_charge_id" to table: "charge_usage_based_revenue_recognitions"
CREATE INDEX "chargeusagebasedrevenuerecognitions_namespace_charge_id" ON "charge_usage_based_revenue_recognitions" ("namespace", "charge_id");
-- create "charge_usage_based_revenue_recognition_credits" table
CREATE TABLE "charge_usage_based_revenue_recognition_credits" (
  "id" character(26) NOT NULL,
  "line_id" character(26) NULL,
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
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_ub_revrec_credits_run" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_revenue_recognitions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeubrevreccredits_ns_run" to table: "charge_usage_based_revenue_recognition_credits"
CREATE INDEX "chargeubrevreccredits_ns_run" ON "charge_usage_based_revenue_recognition_credits" ("namespace", "run_id");
-- create index "chargeusagebasedrevenuerecognitioncredits_annotations" to table: "charge_usage_based_revenue_recognition_credits"
CREATE INDEX "chargeusagebasedrevenuerecognitioncredits_annotations" ON "charge_usage_based_revenue_recognition_credits" USING gin ("annotations");
-- create index "chargeusagebasedrevenuerecognitioncredits_id" to table: "charge_usage_based_revenue_recognition_credits"
CREATE UNIQUE INDEX "chargeusagebasedrevenuerecognitioncredits_id" ON "charge_usage_based_revenue_recognition_credits" ("id");
-- create index "chargeusagebasedrevenuerecognitioncredits_namespace" to table: "charge_usage_based_revenue_recognition_credits"
CREATE INDEX "chargeusagebasedrevenuerecognitioncredits_namespace" ON "charge_usage_based_revenue_recognition_credits" ("namespace");
-- create "charge_usage_based_revenue_recognition_detailed_line" table
CREATE TABLE "charge_usage_based_revenue_recognition_detailed_line" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "annotations" jsonb NULL,
  "currency" character varying NOT NULL,
  "service_period_start" timestamptz NOT NULL,
  "service_period_end" timestamptz NOT NULL,
  "child_unique_reference_id" character varying NOT NULL,
  "index" bigint NULL,
  "quantity" numeric NOT NULL,
  "per_unit_amount" numeric NOT NULL,
  "amount" numeric NOT NULL,
  "amount_discounts" jsonb NULL,
  "credits_applied" jsonb NULL,
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_ub_revrec_detailed_line_run" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_revenue_recognitions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeubrevrecdetailedline_ns_run" to table: "charge_usage_based_revenue_recognition_detailed_line"
CREATE INDEX "chargeubrevrecdetailedline_ns_run" ON "charge_usage_based_revenue_recognition_detailed_line" ("namespace", "run_id");
-- create index "chargeusagebasedrevenuerecognitiondetailedline_annotations" to table: "charge_usage_based_revenue_recognition_detailed_line"
CREATE INDEX "chargeusagebasedrevenuerecognitiondetailedline_annotations" ON "charge_usage_based_revenue_recognition_detailed_line" USING gin ("annotations");
-- create index "chargeusagebasedrevenuerecognitiondetailedline_id" to table: "charge_usage_based_revenue_recognition_detailed_line"
CREATE UNIQUE INDEX "chargeusagebasedrevenuerecognitiondetailedline_id" ON "charge_usage_based_revenue_recognition_detailed_line" ("id");
-- create index "chargeusagebasedrevenuerecognitiondetailedline_namespace" to table: "charge_usage_based_revenue_recognition_detailed_line"
CREATE INDEX "chargeusagebasedrevenuerecognitiondetailedline_namespace" ON "charge_usage_based_revenue_recognition_detailed_line" ("namespace");
-- create index "chargeusagebasedrevenuerecognitiondetailedline_namespace_id" to table: "charge_usage_based_revenue_recognition_detailed_line"
CREATE UNIQUE INDEX "chargeusagebasedrevenuerecognitiondetailedline_namespace_id" ON "charge_usage_based_revenue_recognition_detailed_line" ("namespace", "id");
-- create "charge_usage_based_revenue_recognition_invoiced_usages" table
CREATE TABLE "charge_usage_based_revenue_recognition_invoiced_usages" (
  "id" character(26) NOT NULL,
  "line_id" character(26) NULL,
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
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_ub_revrec_invoiced_usage_run" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_revenue_recognitions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeubrevrecinvoicedusage_ns_run" to table: "charge_usage_based_revenue_recognition_invoiced_usages"
CREATE INDEX "chargeubrevrecinvoicedusage_ns_run" ON "charge_usage_based_revenue_recognition_invoiced_usages" ("namespace", "run_id");
-- create index "chargeusagebasedrevenuerecognitioninvoicedusage_annotations" to table: "charge_usage_based_revenue_recognition_invoiced_usages"
CREATE INDEX "chargeusagebasedrevenuerecognitioninvoicedusage_annotations" ON "charge_usage_based_revenue_recognition_invoiced_usages" USING gin ("annotations");
-- create index "chargeusagebasedrevenuerecognitioninvoicedusage_id" to table: "charge_usage_based_revenue_recognition_invoiced_usages"
CREATE UNIQUE INDEX "chargeusagebasedrevenuerecognitioninvoicedusage_id" ON "charge_usage_based_revenue_recognition_invoiced_usages" ("id");
-- create index "chargeusagebasedrevenuerecognitioninvoicedusage_namespace" to table: "charge_usage_based_revenue_recognition_invoiced_usages"
CREATE INDEX "chargeusagebasedrevenuerecognitioninvoicedusage_namespace" ON "charge_usage_based_revenue_recognition_invoiced_usages" ("namespace");
