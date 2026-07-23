-- create "charge_credit_purchase_cost_bases" table
CREATE TABLE "charge_credit_purchase_cost_bases" (
  "id" character(26) NOT NULL,
  "mode" character varying NOT NULL,
  "fiat_currency" character varying(3) NOT NULL,
  "manual_rate" numeric NULL,
  "resolved_cost_basis" numeric NULL,
  "resolved_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "currency_cost_basis_id" character(26) NULL,
  "resolved_cost_basis_id" character(26) NULL,
  "currency_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_credit_purchase_cost_basis_currency_cost_basis_fk" FOREIGN KEY ("currency_cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "charge_credit_purchase_cost_basis_currency_fk" FOREIGN KEY ("currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "charge_credit_purchase_cost_basis_resolved_cost_basis_fk" FOREIGN KEY ("resolved_cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "fiat_currency_not_empty" CHECK ((fiat_currency)::text <> ''::text),
  CONSTRAINT "resolved_cost_basis_positive" CHECK ((resolved_cost_basis IS NULL) OR (resolved_cost_basis > (0)::numeric)),
  CONSTRAINT "state" CHECK ((((mode)::text = 'dynamic'::text) AND (currency_cost_basis_id IS NULL) AND (manual_rate IS NULL) AND (((resolved_cost_basis_id IS NULL) AND (resolved_cost_basis IS NULL) AND (resolved_at IS NULL)) OR ((resolved_cost_basis_id IS NOT NULL) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)))) OR (((mode)::text = 'pinned'::text) AND (currency_cost_basis_id IS NOT NULL) AND (resolved_cost_basis_id IS NOT NULL) AND (resolved_cost_basis_id = currency_cost_basis_id) AND (manual_rate IS NULL) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)) OR (((mode)::text = 'manual'::text) AND (currency_cost_basis_id IS NULL) AND (resolved_cost_basis_id IS NULL) AND (manual_rate > (0)::numeric) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)))
);
-- create index "chargecreditpurchasecostbasis_currency_cost_basis_id" to table: "charge_credit_purchase_cost_bases"
CREATE INDEX "chargecreditpurchasecostbasis_currency_cost_basis_id" ON "charge_credit_purchase_cost_bases" ("currency_cost_basis_id");
-- create index "chargecreditpurchasecostbasis_currency_id" to table: "charge_credit_purchase_cost_bases"
CREATE INDEX "chargecreditpurchasecostbasis_currency_id" ON "charge_credit_purchase_cost_bases" ("currency_id");
-- create index "chargecreditpurchasecostbasis_id" to table: "charge_credit_purchase_cost_bases"
CREATE UNIQUE INDEX "chargecreditpurchasecostbasis_id" ON "charge_credit_purchase_cost_bases" ("id");
-- create index "chargecreditpurchasecostbasis_namespace" to table: "charge_credit_purchase_cost_bases"
CREATE INDEX "chargecreditpurchasecostbasis_namespace" ON "charge_credit_purchase_cost_bases" ("namespace");
-- create index "chargecreditpurchasecostbasis_resolved_cost_basis_id" to table: "charge_credit_purchase_cost_bases"
CREATE INDEX "chargecreditpurchasecostbasis_resolved_cost_basis_id" ON "charge_credit_purchase_cost_bases" ("resolved_cost_basis_id");
-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "cost_basis_id" character(26) NULL, ADD CONSTRAINT "charge_credit_purchase_cost_basis_charge_fk" FOREIGN KEY ("cost_basis_id") REFERENCES "charge_credit_purchase_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create "charge_flat_fee_cost_bases" table
CREATE TABLE "charge_flat_fee_cost_bases" (
  "id" character(26) NOT NULL,
  "mode" character varying NOT NULL,
  "fiat_currency" character varying(3) NOT NULL,
  "manual_rate" numeric NULL,
  "resolved_cost_basis" numeric NULL,
  "resolved_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "currency_cost_basis_id" character(26) NULL,
  "resolved_cost_basis_id" character(26) NULL,
  "currency_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_cost_basis_currency_cost_basis_fk" FOREIGN KEY ("currency_cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "charge_flat_fee_cost_basis_currency_fk" FOREIGN KEY ("currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "charge_flat_fee_cost_basis_resolved_cost_basis_fk" FOREIGN KEY ("resolved_cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "fiat_currency_not_empty" CHECK ((fiat_currency)::text <> ''::text),
  CONSTRAINT "resolved_cost_basis_positive" CHECK ((resolved_cost_basis IS NULL) OR (resolved_cost_basis > (0)::numeric)),
  CONSTRAINT "state" CHECK ((((mode)::text = 'dynamic'::text) AND (currency_cost_basis_id IS NULL) AND (manual_rate IS NULL) AND (((resolved_cost_basis_id IS NULL) AND (resolved_cost_basis IS NULL) AND (resolved_at IS NULL)) OR ((resolved_cost_basis_id IS NOT NULL) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)))) OR (((mode)::text = 'pinned'::text) AND (currency_cost_basis_id IS NOT NULL) AND (resolved_cost_basis_id IS NOT NULL) AND (resolved_cost_basis_id = currency_cost_basis_id) AND (manual_rate IS NULL) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)) OR (((mode)::text = 'manual'::text) AND (currency_cost_basis_id IS NULL) AND (resolved_cost_basis_id IS NULL) AND (manual_rate > (0)::numeric) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)))
);
-- create index "chargeflatfeecostbasis_currency_cost_basis_id" to table: "charge_flat_fee_cost_bases"
CREATE INDEX "chargeflatfeecostbasis_currency_cost_basis_id" ON "charge_flat_fee_cost_bases" ("currency_cost_basis_id");
-- create index "chargeflatfeecostbasis_currency_id" to table: "charge_flat_fee_cost_bases"
CREATE INDEX "chargeflatfeecostbasis_currency_id" ON "charge_flat_fee_cost_bases" ("currency_id");
-- create index "chargeflatfeecostbasis_id" to table: "charge_flat_fee_cost_bases"
CREATE UNIQUE INDEX "chargeflatfeecostbasis_id" ON "charge_flat_fee_cost_bases" ("id");
-- create index "chargeflatfeecostbasis_namespace" to table: "charge_flat_fee_cost_bases"
CREATE INDEX "chargeflatfeecostbasis_namespace" ON "charge_flat_fee_cost_bases" ("namespace");
-- create index "chargeflatfeecostbasis_resolved_cost_basis_id" to table: "charge_flat_fee_cost_bases"
CREATE INDEX "chargeflatfeecostbasis_resolved_cost_basis_id" ON "charge_flat_fee_cost_bases" ("resolved_cost_basis_id");
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "cost_basis_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fee_cost_basis_charge_fk" FOREIGN KEY ("cost_basis_id") REFERENCES "charge_flat_fee_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create "charge_usage_based_cost_bases" table
CREATE TABLE "charge_usage_based_cost_bases" (
  "id" character(26) NOT NULL,
  "mode" character varying NOT NULL,
  "fiat_currency" character varying(3) NOT NULL,
  "manual_rate" numeric NULL,
  "resolved_cost_basis" numeric NULL,
  "resolved_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "currency_cost_basis_id" character(26) NULL,
  "resolved_cost_basis_id" character(26) NULL,
  "currency_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_cost_basis_currency_cost_basis_fk" FOREIGN KEY ("currency_cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "charge_usage_based_cost_basis_currency_fk" FOREIGN KEY ("currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "charge_usage_based_cost_basis_resolved_cost_basis_fk" FOREIGN KEY ("resolved_cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "fiat_currency_not_empty" CHECK ((fiat_currency)::text <> ''::text),
  CONSTRAINT "resolved_cost_basis_positive" CHECK ((resolved_cost_basis IS NULL) OR (resolved_cost_basis > (0)::numeric)),
  CONSTRAINT "state" CHECK ((((mode)::text = 'dynamic'::text) AND (currency_cost_basis_id IS NULL) AND (manual_rate IS NULL) AND (((resolved_cost_basis_id IS NULL) AND (resolved_cost_basis IS NULL) AND (resolved_at IS NULL)) OR ((resolved_cost_basis_id IS NOT NULL) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)))) OR (((mode)::text = 'pinned'::text) AND (currency_cost_basis_id IS NOT NULL) AND (resolved_cost_basis_id IS NOT NULL) AND (resolved_cost_basis_id = currency_cost_basis_id) AND (manual_rate IS NULL) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)) OR (((mode)::text = 'manual'::text) AND (currency_cost_basis_id IS NULL) AND (resolved_cost_basis_id IS NULL) AND (manual_rate > (0)::numeric) AND (resolved_cost_basis IS NOT NULL) AND (resolved_at IS NOT NULL)))
);
-- create index "chargeusagebasedcostbasis_currency_cost_basis_id" to table: "charge_usage_based_cost_bases"
CREATE INDEX "chargeusagebasedcostbasis_currency_cost_basis_id" ON "charge_usage_based_cost_bases" ("currency_cost_basis_id");
-- create index "chargeusagebasedcostbasis_currency_id" to table: "charge_usage_based_cost_bases"
CREATE INDEX "chargeusagebasedcostbasis_currency_id" ON "charge_usage_based_cost_bases" ("currency_id");
-- create index "chargeusagebasedcostbasis_id" to table: "charge_usage_based_cost_bases"
CREATE UNIQUE INDEX "chargeusagebasedcostbasis_id" ON "charge_usage_based_cost_bases" ("id");
-- create index "chargeusagebasedcostbasis_namespace" to table: "charge_usage_based_cost_bases"
CREATE INDEX "chargeusagebasedcostbasis_namespace" ON "charge_usage_based_cost_bases" ("namespace");
-- create index "chargeusagebasedcostbasis_resolved_cost_basis_id" to table: "charge_usage_based_cost_bases"
CREATE INDEX "chargeusagebasedcostbasis_resolved_cost_basis_id" ON "charge_usage_based_cost_bases" ("resolved_cost_basis_id");
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "cost_basis_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_cost_basis_charge_fk" FOREIGN KEY ("cost_basis_id") REFERENCES "charge_usage_based_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
