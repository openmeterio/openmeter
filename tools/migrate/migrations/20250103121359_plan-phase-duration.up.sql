-- delete all "plan_phases", phases are incompatible with the new plan phase schema
-- atlas:nolint destructive
DELETE FROM "plan_phases";
-- modify "plan_phases" table
-- atlas:nolint destructive data_depend
ALTER TABLE "plan_phases" DROP COLUMN "start_after", ADD COLUMN "index" smallint NOT NULL, ADD COLUMN "duration" character varying NULL;
-- create index "planphase_plan_id_index_deleted_at" to table: "plan_phases"
-- atlas:nolint data_depend
CREATE UNIQUE INDEX "planphase_plan_id_index_deleted_at" ON "plan_phases" ("plan_id", "index", "deleted_at");
