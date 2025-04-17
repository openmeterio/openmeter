-- create index "appcustominvoicing_namespace_id" to table: "app_custom_invoicings"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "appcustominvoicing_namespace_id" ON "app_custom_invoicings" ("namespace", "id");
