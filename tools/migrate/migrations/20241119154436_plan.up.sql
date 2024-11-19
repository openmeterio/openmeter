-- drop index "planphase_plan_id_key" from table: "plan_phases"
DROP INDEX "planphase_plan_id_key";
-- create index "planphase_plan_id_key_deleted_at" to table: "plan_phases"
-- atlas:nolint
CREATE UNIQUE INDEX "planphase_plan_id_key_deleted_at" ON "plan_phases" ("plan_id", "key", "deleted_at");
