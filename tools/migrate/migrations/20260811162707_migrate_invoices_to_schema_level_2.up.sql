BEGIN;

-- Prevent invoice writes while existing customers are migrated to schema level 2.
SELECT *
FROM billing_customer_locks
FOR UPDATE;

CREATE TEMPORARY TABLE om_tmp_billing_schema_level_2_affected_customers (
    customer_id TEXT PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO om_tmp_billing_schema_level_2_affected_customers (customer_id)
SELECT DISTINCT i.customer_id::TEXT
FROM billing_invoices i
WHERE i.schema_level = 1;

SELECT om_func_migrate_customer_invoices_to_schema_level_2_bulk(
    ARRAY(
        SELECT customer_id
        FROM om_tmp_billing_schema_level_2_affected_customers
    )
);

COMMIT;
