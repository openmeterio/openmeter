-- create index "meter_namespace_key" to table: "meters"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "meter_namespace_key" ON "meters" ("namespace", "key") WHERE (deleted_at IS NULL);
