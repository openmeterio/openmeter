-- modify "subscriptions" table
ALTER TABLE
    "subscriptions"
ADD
    COLUMN "is_custom" boolean NOT NULL DEFAULT false;