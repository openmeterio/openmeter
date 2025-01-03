-- delete all "plan_phases"
DELETE FROM "plan_phases";
-- modify "plan_phases" table
ALTER TABLE "plan_phases" DROP COLUMN "start_after", ADD COLUMN "index" smallint NOT NULL, ADD COLUMN "duration" character varying NULL;
-- create index "planphase_plan_id_index_deleted_at" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_index_deleted_at" ON "plan_phases" ("plan_id", "index", "deleted_at");
