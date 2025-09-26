-- Copy back annotations to metadata where it has been updated
UPDATE "grants" SET "metadata" = "annotations" WHERE "metadata" IS DISTINCT FROM "annotations";
