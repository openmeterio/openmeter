-- create index "customersubjects_customer_id" to table: "customer_subjects"
CREATE INDEX "customersubjects_customer_id" ON "customer_subjects" ("customer_id");
-- create index "customersubjects_deleted_at" to table: "customer_subjects"
CREATE INDEX "customersubjects_deleted_at" ON "customer_subjects" ("deleted_at");
-- create index "customersubjects_deleted_at_customer_id" to table: "customer_subjects"
CREATE INDEX "customersubjects_deleted_at_customer_id" ON "customer_subjects" ("deleted_at", "customer_id");
-- create index "customersubjects_subject_key" to table: "customer_subjects"
CREATE INDEX "customersubjects_subject_key" ON "customer_subjects" ("subject_key");
