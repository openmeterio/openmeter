-- create index "customersubjects_subject_key_trgm" to table: "customer_subjects"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "customersubjects_subject_key_trgm" ON "customer_subjects" USING gin ("subject_key" gin_trgm_ops);
