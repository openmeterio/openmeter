-- reverse: create index "customersubjects_namespace_subject_key" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key";

-- reverse: create index "customersubjects_namespace" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace";

-- reverse: modify "customer_subjects" table
ALTER TABLE
    "customer_subjects" DROP CONSTRAINT "customer_subjects_customers_subjects",
    DROP COLUMN "namespace",
ADD
    CONSTRAINT "customer_subjects_customers_subjects" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- reverse: modify "customers" table
--
--
-- manual interventions as we're breaking lint rules
-- atlas:nolint MF103
ALTER TABLE
    "customers"
ADD
    COLUMN "key" character varying NOT NULL;

CREATE UNIQUE INDEX "customer_namespace_key_deleted_at" ON "customers" ("namespace", "key", "deleted_at");