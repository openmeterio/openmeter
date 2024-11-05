-- create "plans" table
CREATE TABLE "plans" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "version" bigint NOT NULL,
  "currency" character varying NOT NULL DEFAULT 'USD',
  "effective_from" timestamptz NULL,
  "effective_to" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "plan_id" to table: "plans"
CREATE UNIQUE INDEX "plan_id" ON "plans" ("id");
-- create index "plan_namespace" to table: "plans"
CREATE INDEX "plan_namespace" ON "plans" ("namespace");
-- create index "plan_namespace_id" to table: "plans"
CREATE UNIQUE INDEX "plan_namespace_id" ON "plans" ("namespace", "id");
-- create index "plan_namespace_key_deleted_at" to table: "plans"
CREATE UNIQUE INDEX "plan_namespace_key_deleted_at" ON "plans" ("namespace", "key", "deleted_at");
-- create index "plan_namespace_key_version" to table: "plans"
CREATE UNIQUE INDEX "plan_namespace_key_version" ON "plans" ("namespace", "key", "version");
-- create "plan_phases" table
CREATE TABLE "plan_phases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "start_after" character varying NOT NULL DEFAULT 'P0D',
  "discounts" jsonb NULL,
  "plan_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_phases_plans_phases" FOREIGN KEY ("plan_id") REFERENCES "plans" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "planphase_id" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_id" ON "plan_phases" ("id");
-- create index "planphase_namespace" to table: "plan_phases"
CREATE INDEX "planphase_namespace" ON "plan_phases" ("namespace");
-- create index "planphase_namespace_id" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_namespace_id" ON "plan_phases" ("namespace", "id");
-- create index "planphase_namespace_key" to table: "plan_phases"
CREATE INDEX "planphase_namespace_key" ON "plan_phases" ("namespace", "key");
-- create index "planphase_namespace_key_deleted_at" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_namespace_key_deleted_at" ON "plan_phases" ("namespace", "key", "deleted_at");
-- create index "planphase_plan_id_key" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_key" ON "plan_phases" ("plan_id", "key");
-- create "plan_rate_cards" table
CREATE TABLE "plan_rate_cards" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "type" character varying NOT NULL,
  "feature_key" character varying NULL,
  "entitlement_template" jsonb NULL,
  "tax_config" jsonb NULL,
  "billing_cadence" character varying NULL,
  "price" jsonb NULL,
  "feature_id" character(26) NULL,
  "phase_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_rate_cards_features_ratecard" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "plan_rate_cards_plan_phases_ratecards" FOREIGN KEY ("phase_id") REFERENCES "plan_phases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "planratecard_id" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_id" ON "plan_rate_cards" ("id");
-- create index "planratecard_namespace" to table: "plan_rate_cards"
CREATE INDEX "planratecard_namespace" ON "plan_rate_cards" ("namespace");
-- create index "planratecard_namespace_id" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_namespace_id" ON "plan_rate_cards" ("namespace", "id");
-- create index "planratecard_namespace_key_deleted_at" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_namespace_key_deleted_at" ON "plan_rate_cards" ("namespace", "key", "deleted_at");
-- create index "planratecard_phase_id_feature_key" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_feature_key" ON "plan_rate_cards" ("phase_id", "feature_key");
-- create index "planratecard_phase_id_key" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_key" ON "plan_rate_cards" ("phase_id", "key");
