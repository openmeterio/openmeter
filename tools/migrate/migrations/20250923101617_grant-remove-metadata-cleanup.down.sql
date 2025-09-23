-- reverse: modify "grants" table
ALTER TABLE "grants" ADD COLUMN "metadata" jsonb NULL;
