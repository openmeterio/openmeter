-- reverse: create index "planphase_plan_id_key_deleted_at" to table: "plan_phases"
DROP INDEX "planphase_plan_id_key_deleted_at";
-- reverse: drop index "planphase_plan_id_key" from table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_key" ON "plan_phases" ("plan_id", "key");
