-- delete all "plan_phases"
DELETE FROM "plan_phases";
-- reverse: create index "planphase_plan_id_index_deleted_at" to table: "plan_phases"
DROP INDEX "planphase_plan_id_index_deleted_at";
-- reverse: modify "plan_phases" table
ALTER TABLE "plan_phases" DROP COLUMN "duration", DROP COLUMN "index", ADD COLUMN "start_after" character varying NOT NULL DEFAULT 'P0D';
