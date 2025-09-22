-- reverse: modify "grants" table
ALTER TABLE "grants" ALTER COLUMN "expires_at" SET NOT NULL, ALTER COLUMN "expiration" SET NOT NULL;
