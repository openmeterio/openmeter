-- reverse: modify "apps" table
ALTER TABLE "apps" ADD COLUMN "is_default" boolean NOT NULL DEFAULT false;
