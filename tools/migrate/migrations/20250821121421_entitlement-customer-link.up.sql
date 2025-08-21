-- drop index "entitlement_namespace_id_subject_key" from table: "entitlements"
DROP INDEX IF EXISTS "entitlement_namespace_id_subject_key";

-- 1) add column as NULLABLE first so we can backfill
ALTER TABLE "entitlements" ADD COLUMN "customer_id" character(26);

-- 2) backfill customer_id from usage attribution mapping (customer_subjects)
--    We match by namespace + subject_key and only consider active (deleted_at IS NULL) mappings
UPDATE "entitlements" e
SET "customer_id" = cs."customer_id"
FROM "customer_subjects" cs
WHERE cs."namespace" = e."namespace"
  AND cs."subject_key" = e."subject_key"
  AND cs."deleted_at" IS NULL
  AND (e."customer_id" IS NULL OR e."customer_id" = '');

-- 3) add foreign key constraint (allow NULLs for legacy rows)
ALTER TABLE "entitlements"
  ADD CONSTRAINT "entitlements_customers_entitlements"
  FOREIGN KEY ("customer_id") REFERENCES "customers" ("id")
  ON UPDATE NO ACTION ON DELETE NO ACTION;

-- 4) indexes
CREATE INDEX IF NOT EXISTS "entitlement_namespace_customer_id" ON "entitlements" ("namespace", "customer_id");
CREATE INDEX IF NOT EXISTS "entitlement_namespace_id_customer_id" ON "entitlements" ("namespace", "id", "customer_id");
CREATE INDEX IF NOT EXISTS "entitlement_namespace_subject_id" ON "entitlements" ("namespace", "subject_id");

-- 5) make customer_id NOT NULL
-- atlas:nolint MF104
ALTER TABLE "entitlements" ALTER COLUMN "customer_id" SET NOT NULL;
