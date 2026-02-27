-- create "charge_credit_realizations" table
CREATE TABLE "charge_credit_realizations" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "charge_id" character(26) NOT NULL,
  "std_realization_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_credit_realizations_charges_credit_realizations" FOREIGN KEY ("charge_id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "charge_credit_realizations_standard_invoice_settlements_credit_" FOREIGN KEY ("std_realization_id") REFERENCES "standard_invoice_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "chargecreditrealization_annotations" to table: "charge_credit_realizations"
CREATE INDEX "chargecreditrealization_annotations" ON "charge_credit_realizations" USING gin ("annotations");
-- create index "chargecreditrealization_id" to table: "charge_credit_realizations"
CREATE UNIQUE INDEX "chargecreditrealization_id" ON "charge_credit_realizations" ("id");
-- create index "chargecreditrealization_namespace" to table: "charge_credit_realizations"
CREATE INDEX "chargecreditrealization_namespace" ON "charge_credit_realizations" ("namespace");
