-- modify "plans" table
ALTER TABLE "plans" ADD COLUMN "is_custom" boolean NOT NULL DEFAULT false;
