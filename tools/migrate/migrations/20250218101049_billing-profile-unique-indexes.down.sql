-- reverse: create index "billingprofile_namespace_default" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default";
-- reverse: drop index "billingprofile_namespace_default_deleted_at" from table: "billing_profiles"
CREATE UNIQUE INDEX "billingprofile_namespace_default_deleted_at" ON "billing_profiles" ("namespace", "default", "deleted_at");
