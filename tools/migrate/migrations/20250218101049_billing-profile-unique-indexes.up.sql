-- drop index "billingprofile_namespace_default_deleted_at" from table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default_deleted_at";
-- create index "billingprofile_namespace_default" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingprofile_namespace_default" ON "billing_profiles" ("namespace", "default") WHERE (deleted_at IS NULL);
