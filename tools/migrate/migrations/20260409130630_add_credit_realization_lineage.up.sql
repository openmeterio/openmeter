-- create "credit_realization_lineages" table
CREATE TABLE "credit_realization_lineages" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "root_realization_id" character(26) NOT NULL,
  "customer_id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "origin_kind" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "creditrealizationlineage_id" to table: "credit_realization_lineages"
CREATE UNIQUE INDEX "creditrealizationlineage_id" ON "credit_realization_lineages" ("id");
-- create index "creditrealizationlineage_namespace" to table: "credit_realization_lineages"
CREATE INDEX "creditrealizationlineage_namespace" ON "credit_realization_lineages" ("namespace");
-- create index "creditrealizationlineage_namespace_customer_id" to table: "credit_realization_lineages"
CREATE INDEX "creditrealizationlineage_namespace_customer_id" ON "credit_realization_lineages" ("namespace", "customer_id");
-- create index "creditrealizationlineage_namespace_root_realization_id" to table: "credit_realization_lineages"
CREATE UNIQUE INDEX "creditrealizationlineage_namespace_root_realization_id" ON "credit_realization_lineages" ("namespace", "root_realization_id");
-- create "credit_realization_lineage_segments" table
CREATE TABLE "credit_realization_lineage_segments" (
  "id" character(26) NOT NULL,
  "amount" numeric NOT NULL,
  "state" character varying NOT NULL,
  "backing_transaction_group_id" character(26) NULL,
  "closed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "lineage_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "credit_realization_lineage_segments_credit_realization_lineages" FOREIGN KEY ("lineage_id") REFERENCES "credit_realization_lineages" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "creditrealizationlineagesegment_id" to table: "credit_realization_lineage_segments"
CREATE UNIQUE INDEX "creditrealizationlineagesegment_id" ON "credit_realization_lineage_segments" ("id");
-- create index "creditrealizationlineagesegment_lineage_id" to table: "credit_realization_lineage_segments"
CREATE INDEX "creditrealizationlineagesegment_lineage_id" ON "credit_realization_lineage_segments" ("lineage_id");
-- create index "creditrealizationlineagesegment_lineage_id_closed_at" to table: "credit_realization_lineage_segments"
CREATE INDEX "creditrealizationlineagesegment_lineage_id_closed_at" ON "credit_realization_lineage_segments" ("lineage_id", "closed_at");
