-- drop index "plan_namespace_key_version" from table: "plans"
DROP INDEX "plan_namespace_key_version";
-- create index "plan_namespace_key_version_deleted_at" to table: "plans"
-- atlas:nolint
CREATE UNIQUE INDEX "plan_namespace_key_version_deleted_at" ON "plans" ("namespace", "key", "version", "deleted_at");
