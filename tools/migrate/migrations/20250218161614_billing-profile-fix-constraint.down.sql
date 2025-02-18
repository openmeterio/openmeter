-- reverse: create index "billingprofile_namespace_default" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default";
-- reverse: drop index "billingprofile_namespace_default" from table: "billing_profiles"
CREATE UNIQUE INDEX "billingprofile_namespace_default" ON "billing_profiles" ("namespace", "default") WHERE (deleted_at IS NULL);
