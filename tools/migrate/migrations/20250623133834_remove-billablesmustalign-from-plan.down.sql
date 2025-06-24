-- reverse: modify "plans" table
ALTER TABLE "plans" ADD COLUMN "billables_must_align" boolean NOT NULL DEFAULT false;
