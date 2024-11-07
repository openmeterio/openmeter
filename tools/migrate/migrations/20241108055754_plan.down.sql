-- reverse: create index "plan_namespace_key_version_deleted_at" to table: "plans"
DROP INDEX "plan_namespace_key_version_deleted_at";
-- reverse: drop index "plan_namespace_key_version" from table: "plans"
CREATE UNIQUE INDEX "plan_namespace_key_version" ON "plans" ("namespace", "key", "version");
