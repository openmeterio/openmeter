-- modify "addons" table
ALTER TABLE "addons" ADD COLUMN "instance_type" character varying NOT NULL DEFAULT 'single';
