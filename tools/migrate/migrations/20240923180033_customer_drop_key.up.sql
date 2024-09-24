-- modify "customers" table
-- atlas:nolint DS103
ALTER TABLE
    "customers" DROP COLUMN "key";

-- modify "customer_subjects" table
-- atlas:nolint MF103
ALTER TABLE
    "customer_subjects" DROP CONSTRAINT "customer_subjects_customers_subjects",
ADD
    COLUMN "namespace" character varying NOT NULL,
ADD
    CONSTRAINT "customer_subjects_customers_subjects" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;

-- create index "customersubjects_namespace" to table: "customer_subjects"
-- atlas:nolint MF103
CREATE INDEX "customersubjects_namespace" ON "customer_subjects" ("namespace");

-- create index "customersubjects_namespace_subject_key" to table: "customer_subjects"
--
--
-- manual interventions as we're breaking lint rules
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key" ON "customer_subjects" ("namespace", "subject_key");