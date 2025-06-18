-- reverse: create index "app_namespace_type_is_default" to table: "apps"
DROP INDEX "app_namespace_type_is_default";
-- atlas:nolint MF101
-- reverse: drop index "app_namespace_type_is_default" from table: "apps"
CREATE UNIQUE INDEX "app_namespace_type_is_default" ON "apps" ("namespace", "type", "is_default") WHERE (is_default = true);
