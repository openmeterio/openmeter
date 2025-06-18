-- reverse: create index "app_namespace_type_is_default" to table: "apps"
DROP INDEX "app_namespace_type_is_default";
-- reverse: modify "apps" table
-- atlas:nolint DS103
ALTER TABLE "apps" DROP COLUMN "is_default";
