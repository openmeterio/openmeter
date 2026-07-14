-- Verify every database object that is required before recording baseline 20260709134422.
DO $$
BEGIN
    IF to_regclass('charges_search_v1s') IS NULL THEN
        RAISE EXCEPTION 'legacy Ent reconciliation did not create charges_search_v1s';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'om_func_migrate_customer_invoices_to_schema_level_2') THEN
        RAISE EXCEPTION 'legacy Ent reconciliation did not create the invoice schema migration function';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'om_func_update_usage_period_durations_batch') THEN
        RAISE EXCEPTION 'legacy Ent reconciliation did not create the usage period migration functions';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trigger_delete_entitlement_on_subscription_item_delete'
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION 'legacy Ent reconciliation did not create the subscription item delete trigger';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM billing_invoice_write_schema_levels
        WHERE id = 'write_schema_level'
          AND schema_level = 1
    ) THEN
        RAISE EXCEPTION 'legacy Ent reconciliation did not initialize the invoice write schema level';
    END IF;
END
$$;
