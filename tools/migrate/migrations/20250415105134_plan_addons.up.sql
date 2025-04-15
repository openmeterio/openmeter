-- create "plan_addons" table
CREATE TABLE "plan_addons" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "from_plan_phase" character varying NOT NULL,
  "max_quantity" bigint NULL,
  "addon_id" character(26) NOT NULL,
  "plan_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_addons_addons_plans" FOREIGN KEY ("addon_id") REFERENCES "addons" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plan_addons_plans_addons" FOREIGN KEY ("plan_id") REFERENCES "plans" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "planaddon_annotations" to table: "plan_addons"
CREATE INDEX "planaddon_annotations" ON "plan_addons" USING gin ("annotations");
-- create index "planaddon_id" to table: "plan_addons"
CREATE UNIQUE INDEX "planaddon_id" ON "plan_addons" ("id");
-- create index "planaddon_namespace" to table: "plan_addons"
CREATE INDEX "planaddon_namespace" ON "plan_addons" ("namespace");
-- create index "planaddon_namespace_plan_id_addon_id" to table: "plan_addons"
CREATE UNIQUE INDEX "planaddon_namespace_plan_id_addon_id" ON "plan_addons" ("namespace", "plan_id", "addon_id") WHERE (deleted_at IS NULL);
