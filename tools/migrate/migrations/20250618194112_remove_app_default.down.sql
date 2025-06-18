-- reverse: modify "apps" table
ALTER TABLE "apps" ADD COLUMN "is_default" boolean NOT NULL DEFAULT false;

-- atlas:nolint MF101
-- reverse: drop index "app_namespace_type_is_default" from table: "apps"
CREATE UNIQUE INDEX "app_namespace_type_is_default" ON "apps" ("namespace", "type", "is_default") WHERE (is_default = true);
