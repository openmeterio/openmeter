-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "billables_must_align" boolean NOT NULL DEFAULT false;
