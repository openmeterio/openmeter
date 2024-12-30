-- modify "plan_phases" table
ALTER TABLE "plan_phases" DROP COLUMN "start_after", ADD COLUMN "duration" character varying NULL, ADD COLUMN "index" bigint NOT NULL;
-- create index "planphase_plan_id_index_deleted_at" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_index_deleted_at" ON "plan_phases" ("plan_id", "index", "deleted_at");
