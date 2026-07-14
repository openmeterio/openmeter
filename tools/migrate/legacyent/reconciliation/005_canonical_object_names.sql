-- Frozen reconciliation for Ent baseline 20260709134422.
-- Ent and Atlas derive different names for long constraints and indexes. Future
-- Atlas migrations must see the canonical names produced by a from-scratch run.
DO $$
DECLARE
    object_name RECORD;
    target_schema TEXT := current_schema();
BEGIN
    FOR object_name IN
        SELECT *
        FROM (VALUES
            ('app_custom_invoicing_customers', 'app_custom_invoicing_customers_ada5da889315de7a30fe8beef569f822', 'app_custom_invoicing_customers_app_custom_invoicings_customer_a'),
            ('billing_customer_overrides', 'billing_customer_overrides_bil_5c017296bc262fb5fccf780bdcfca6c4', 'billing_customer_overrides_billing_profiles_billing_customer_ov'),
            ('billing_invoice_line_discounts', 'billing_invoice_line_discounts_11cf9324afed211e97983a4e9519dbe1', 'billing_invoice_line_discounts_billing_invoice_lines_line_amoun'),
            ('billing_invoice_line_usage_discounts', 'billing_invoice_line_usage_dis_0ad728bf5a151035e65a2ce1afd52899', 'billing_invoice_line_usage_discounts_billing_invoice_lines_line'),
            ('billing_invoice_lines', 'billing_invoice_lines_billing__9051cb6caf60cd672d88905d404644da', 'billing_invoice_lines_billing_invoice_flat_fee_line_configs_fla'),
            ('billing_invoice_lines', 'billing_invoice_lines_billing__b027743b4950708c812ebd2438bac5b1', 'billing_invoice_lines_billing_invoice_split_line_groups_billing'),
            ('billing_invoice_lines', 'billing_invoice_lines_billing__5450c6e047657ba77d816bb17cfec392', 'billing_invoice_lines_billing_invoice_usage_based_line_configs_'),
            ('billing_invoice_split_line_groups', 'billing_invoice_split_line_gro_9e09b8a07f518f8398a34633d0cac332', 'billing_invoice_split_line_groups_charges_billing_split_line_gr'),
            ('billing_invoice_split_line_groups', 'billing_invoice_split_line_gro_c63de4007ced9bf89bfd7a928b4d9692', 'billing_invoice_split_line_groups_subscriptions_billing_split_l'),
            ('billing_invoice_split_line_groups', 'billing_invoice_split_line_gro_b88cdc0ad31c1df8455986347382de49', 'billing_invoice_split_line_groups_subscription_items_billing_sp'),
            ('billing_invoice_split_line_groups', 'billing_invoice_split_line_gro_f1be3bc887c06ba0bf729db84bb2c1cb', 'billing_invoice_split_line_groups_subscription_phases_billing_s'),
            ('billing_invoice_validation_issues', 'billing_invoice_validation_iss_553826c8285b0b750248685781a429f5', 'billing_invoice_validation_issues_billing_invoices_billing_invo'),
            ('billing_standard_invoice_detailed_line_amount_discounts', 'billing_standard_invoice_detai_02cbd055835995b9d8b2a22bb07b34be', 'billing_standard_invoice_detailed_line_amount_discounts_billing'),
            ('billing_standard_invoice_detailed_lines', 'billing_standard_invoice_detai_16b3b6a066b9167ffe040a46046bdbb3', 'billing_standard_invoice_detailed_lines_billing_invoices_billin'),
            ('billing_standard_invoice_detailed_lines', 'billing_standard_invoice_detai_afe9d2242b4b553d7970384d63e74b6e', 'billing_standard_invoice_detailed_lines_billing_invoice_lines_d'),
            ('charge_credit_purchase_credit_grants', 'charge_credit_purchase_credit__d988851d507ef1e1146e47fbeb4d8e7a', 'charge_credit_purchase_credit_grants_charge_credit_purchases_cr'),
            ('charge_credit_purchase_external_payments', 'charge_credit_purchase_externa_ba1c9a44b0bdc963e88dad74f95c287d', 'charge_credit_purchase_external_payments_charge_credit_purchase'),
            ('charge_credit_purchase_invoiced_payments', 'charge_credit_purchase_invoice_bc4726b6cd5c451649bc81ed0af8b849', 'charge_credit_purchase_invoiced_payments_charge_credit_purchase'),
            ('charge_credit_purchase_invoiced_payments', 'charge_credit_purchase_invoice_5451d50bc2c75c3f54a54c2d4ec59dde', 'charge_credit_purchase_invoiced_payments_billing_invoice_lines_'),
            ('charge_credit_purchases', 'charge_credit_purchases_subscr_8ece17f6a26cc18c16d8472b26c3c78a', 'charge_credit_purchases_subscription_items_charges_credit_purch'),
            ('charge_credit_purchases', 'charge_credit_purchases_subscr_11251e075079165d7a55ab829424182c', 'charge_credit_purchases_subscription_phases_charges_credit_purc'),
            ('charge_flat_fee_run_credit_allocations', 'charge_flat_fee_run_credit_all_93b746677fff18e39c64917bf17b23c7', 'charge_flat_fee_run_credit_allocations_charge_flat_fee_run_cred'),
            ('charge_flat_fee_run_credit_allocations', 'charge_flat_fee_run_credit_all_870149540e706975bf9cae457fd583c1', 'charge_flat_fee_run_credit_allocations_billing_invoice_lines_ch'),
            ('charge_flat_fee_run_detailed_lines', 'charge_flat_fee_run_detailed_l_0fa1a4f63a631e0465b4ed8b64df8e38', 'charge_flat_fee_run_detailed_lines_charge_flat_fee_runs_detaile'),
            ('charge_flat_fee_run_invoiced_usages', 'charge_flat_fee_run_invoiced_u_62e50dcc3bc0ad28771b6c72510fac8f', 'charge_flat_fee_run_invoiced_usages_charge_flat_fee_runs_invoic'),
            ('charge_flat_fee_run_payments', 'charge_flat_fee_run_payments_b_4000652989835922f2c767af7d4db0c4', 'charge_flat_fee_run_payments_billing_invoice_lines_charge_flat_'),
            ('charge_usage_based_overrides', 'charge_usage_based_overrides_t_4c4a2e408f2c1b7a27cafd13cd83641e', 'charge_usage_based_overrides_tax_codes_charge_usage_based_overr'),
            ('charge_usage_based_run_credit_allocations', 'charge_usage_based_run_credit__206fce521944e896718bd6301b4da756', 'charge_usage_based_run_credit_allocations_charge_usage_based_ru'),
            ('charge_usage_based_run_detailed_line', 'charge_usage_based_run_detaile_2fee50b0cfa3bfec48f9c3a6e6eead6f', 'charge_usage_based_run_detailed_line_charge_usage_based_detaile'),
            ('charge_usage_based_run_detailed_line', 'charge_usage_based_run_detaile_42e8ecc9a202ee71a0a0666f0cff0052', 'charge_usage_based_run_detailed_line_charge_usage_based_runs_de'),
            ('charge_usage_based_run_invoiced_usages', 'charge_usage_based_run_invoice_fd0dd4b4e43a8a817ddb78c737805bdb', 'charge_usage_based_run_invoiced_usages_charge_usage_based_runs_'),
            ('charge_usage_based_runs', 'charge_usage_based_runs_billin_1c215cc3089168133502501cb37efbe3', 'charge_usage_based_runs_billing_invoices_charge_usage_based_run'),
            ('charge_usage_based_runs', 'charge_usage_based_runs_billin_f8043f7a82b84a9210b7a583dc4ceeac', 'charge_usage_based_runs_billing_invoice_lines_charge_usage_base'),
            ('credit_realization_lineage_segments', 'credit_realization_lineage_seg_f2dfecda12e4514dfd737cd127b67014', 'credit_realization_lineage_segments_credit_realization_lineages'),
            ('ledger_breakage_records', 'ledger_breakage_records_ledger_2c14135883ebbab796172865d2bb2ebd', 'ledger_breakage_records_ledger_transaction_groups_breakage_reco'),
            ('ledger_breakage_records', 'ledger_breakage_records_ledger_c6b2597292e0d8b5c227e69317d9ac66', 'ledger_breakage_records_ledger_sub_accounts_fbo_breakage_record'),
            ('ledger_breakage_records', 'ledger_breakage_records_ledger_015f8aabf0e937bbcd36c821a8a8ad8e', 'ledger_breakage_records_ledger_breakage_records_planned_release'),
            ('ledger_breakage_records', 'ledger_breakage_records_ledger_d72eba96776f77a4a624e6305179ef55', 'ledger_breakage_records_ledger_transaction_groups_source_breaka'),
            ('ledger_breakage_records', 'ledger_breakage_records_ledger_5127242db4cf0c63d4781cc55fa56076', 'ledger_breakage_records_ledger_transactions_source_breakage_rec'),
            ('notification_event_delivery_status_events', 'notification_event_delivery_st_6eec16a871d896e0cf1242bb186b54bf', 'notification_event_delivery_status_events_notification_event_de'),
            ('subscription_billing_sync_states', 'subscription_billing_sync_stat_718158a45c2ea1f8277ec880deb7a4c9', 'subscription_billing_sync_states_subscriptions_billing_sync_sta')
        ) AS names(table_name, ent_name, atlas_name)
    LOOP
        IF EXISTS (
            SELECT 1
            FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = t.relnamespace
            WHERE n.nspname = target_schema
              AND t.relname = object_name.table_name
              AND c.conname = object_name.ent_name
        ) AND NOT EXISTS (
            SELECT 1
            FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = t.relnamespace
            WHERE n.nspname = target_schema
              AND t.relname = object_name.table_name
              AND c.conname = object_name.atlas_name
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.%I RENAME CONSTRAINT %I TO %I',
                target_schema,
                object_name.table_name,
                object_name.ent_name,
                object_name.atlas_name
            );
        END IF;
    END LOOP;

    FOR object_name IN
        SELECT *
        FROM (VALUES
            ('billing_invoice_line_discounts', 'billinginvoicelinediscount_nam_2d0529871a28989e56a515ec143311fe', 'billinginvoicelinediscount_namespace_line_id_child_unique_refer'),
            ('billing_invoice_line_usage_discounts', 'billinginvoicelineusagediscoun_21c13af2195768635d6700e9670b8d24', 'billinginvoicelineusagediscount_namespace_line_id_child_unique_'),
            ('billing_invoice_lines', 'billinginvoiceline_namespace_s_71b0f19e447b0f048a1531365e50ce65', 'billinginvoiceline_namespace_subscription_id_subscription_phase'),
            ('billing_invoice_lines', 'billinginvoiceline_namespace_p_19b1f479440f0f08a5f01eaaf9fc204b', 'billinginvoiceline_namespace_parent_line_id_child_unique_refere'),
            ('billing_standard_invoice_detailed_line_amount_discounts', 'billingstandardinvoicedetailed_fa7b94c971930b9cdf6fd11d79204285', 'billingstandardinvoicedetailedlineamountdiscount_namespace_line'),
            ('currency_cost_bases', 'currencycostbasis_namespace_cu_fc6e8d08234082a53c818401f36cf0b0', 'currencycostbasis_namespace_currency_id_fiat_code_effective_fro'),
            ('ledger_sub_account_routes', 'ledgersubaccountroute_namespac_f652e8291f52ed4eee30611d43d3e264', 'ledgersubaccountroute_namespace_account_id_routing_key_version_')
        ) AS names(table_name, ent_name, atlas_name)
    LOOP
        IF to_regclass(format('%I.%I', target_schema, object_name.ent_name)) IS NOT NULL
           AND to_regclass(format('%I.%I', target_schema, object_name.atlas_name)) IS NULL THEN
            EXECUTE format(
                'ALTER INDEX %I.%I RENAME TO %I',
                target_schema,
                object_name.ent_name,
                object_name.atlas_name
            );
        END IF;
    END LOOP;
END
$$;
