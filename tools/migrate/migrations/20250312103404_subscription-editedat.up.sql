-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "last_edited_at" timestamptz NULL;
