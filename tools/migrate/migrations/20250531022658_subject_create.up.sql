-- create "subjects" table
CREATE TABLE "subjects" (
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "current_period_start" timestamptz NULL,
    "current_period_end" timestamptz NULL,
    "id" character(26) NOT NULL,
    "key" character varying NOT NULL,
    "display_name" character varying NULL,
    "metadata" jsonb NULL,
    "stripe_customer_id" character varying NULL,
    "organization_subjects" character varying NOT NULL,
    PRIMARY KEY ("id")
);

-- create index "subject_id" to table: "subjects"
CREATE UNIQUE INDEX "subject_id" ON "subjects" ("id");

-- create index "subject_key_organization_subjects" to table: "subjects"
CREATE UNIQUE INDEX "subject_key_organization_subjects" ON "subjects" ("key", "organization_subjects");

-- create index "subject_display_name" to table: "subjects"
CREATE INDEX "subject_display_name" ON "subjects" ("display_name");