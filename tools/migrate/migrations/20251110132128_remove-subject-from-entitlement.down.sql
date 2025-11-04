-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "subject_id" character(26) NOT NULL, ADD COLUMN "subject_key" character varying NOT NULL;

-- lets add back those indexes atlas somehow misses....
CREATE INDEX IF NOT EXISTS "entitlement_namespace_subject_key" ON "entitlements" ("namespace", "subject_key");
CREATE INDEX IF NOT EXISTS "entitlement_namespace_subject_id" ON "entitlements" ("namespace", "subject_id");

-- and the constraint
ALTER TABLE "entitlements" ADD CONSTRAINT "entitlements_subjects_entitlements" FOREIGN KEY ("subject_id") REFERENCES "subjects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
