BEGIN;

-- Prevent invoice writes while existing customers are migrated to schema level 2.
SELECT *
FROM billing_customer_locks
FOR UPDATE;

CREATE TEMPORARY TABLE om_tmp_billing_schema_level_2_affected_customers (
    customer_id TEXT PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO om_tmp_billing_schema_level_2_affected_customers (customer_id)
SELECT DISTINCT l.customer_id::TEXT
FROM billing_customer_locks l
JOIN billing_invoices i
    ON i.namespace = l.namespace
    AND i.customer_id = l.customer_id
WHERE i.schema_level = 1;

SELECT om_func_migrate_customer_invoices_to_schema_level_2_bulk(
    ARRAY(
        SELECT customer_id
        FROM om_tmp_billing_schema_level_2_affected_customers
    )
);

COMMIT;
