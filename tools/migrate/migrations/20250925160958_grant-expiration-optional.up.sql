-- modify "grants" table
ALTER TABLE "grants" ALTER COLUMN "expiration" DROP NOT NULL, ALTER COLUMN "expires_at" DROP NOT NULL;
