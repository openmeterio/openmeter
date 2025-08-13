-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" DROP CONSTRAINT "entitlements_subjects_entitlements", DROP COLUMN "subject_id";
