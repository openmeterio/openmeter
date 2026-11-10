-- gin_trgm_ops requires pg_trgm. Atlas does not manage extensions, so this is added by hand.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- create index "customersubjects_subject_key_trgm" to table: "customer_subjects"
CREATE INDEX "customersubjects_subject_key_trgm" ON "customer_subjects" USING gin ("subject_key" gin_trgm_ops);
