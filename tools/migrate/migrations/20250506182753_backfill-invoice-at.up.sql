UPDATE "billing_invoices" SET "issued_at" = "updated_at" WHERE "issued_at" IS NULL AND "status" = 'issued';
UPDATE "billing_invoices" SET "issued_at" = "updated_at" WHERE "issued_at" IS NULL AND "status" LIKE 'payment_processing.%';
UPDATE "billing_invoices" SET "issued_at" = "updated_at" WHERE "issued_at" IS NULL AND "status" = 'paid';
UPDATE "billing_invoices" SET "issued_at" = "updated_at" WHERE "issued_at" IS NULL AND "status" = 'overdue';
UPDATE "billing_invoices" SET "issued_at" = "updated_at" WHERE "issued_at" IS NULL AND "status" = 'uncollectible';
UPDATE "billing_invoices" SET "issued_at" = "updated_at" WHERE "issued_at" IS NULL AND "status" = 'voided';
