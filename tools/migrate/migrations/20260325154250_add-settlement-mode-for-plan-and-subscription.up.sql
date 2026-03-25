-- modify "plans" table
ALTER TABLE "plans" ADD COLUMN "settlement_mode" character varying NOT NULL DEFAULT 'credit_then_invoice';
-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "settlement_mode" character varying NOT NULL DEFAULT 'credit_then_invoice';
