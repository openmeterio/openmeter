-- create index "plan_namespace_updated_at" to table: "plans"
CREATE INDEX "plan_namespace_updated_at" ON "plans" ("namespace", "updated_at");
