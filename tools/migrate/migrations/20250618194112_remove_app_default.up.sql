-- This migration removes the "is_default" column from the "apps" table
-- drop index "app_namespace_type_is_default" from table: "apps"
DROP INDEX IF EXISTS "app_namespace_type_is_default";

-- modify "apps" table
ALTER TABLE "apps" DROP COLUMN "is_default";
