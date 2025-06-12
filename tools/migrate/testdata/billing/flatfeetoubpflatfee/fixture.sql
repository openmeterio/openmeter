--
-- PostgreSQL database dump
--

-- Dumped from database version 14.17
-- Dumped by pg_dump version 15.13 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: addons; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: features; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.features (id, created_at, updated_at, deleted_at, metadata, namespace, name, key, meter_slug, meter_group_by_filters, archived_at) VALUES ('01JWB2ND12J6DMTS4T0RCFSQH3', '2025-05-28 09:13:06.850727+00', '2025-05-28 09:13:06.850727+00', NULL, 'null', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'api-requests-total', 'api-requests-total', 'api-requests-total', NULL, NULL);


--
-- Data for Name: addon_rate_cards; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: apps; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.apps (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, type, status, is_default) VALUES ('01JWB2ND0MDGNC6WZBBW8ZQCAC', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', NULL, '2025-05-28 09:13:06.836674+00', '2025-05-28 09:13:06.836675+00', NULL, 'Sandbox', 'Sandbox app', 'sandbox', 'ready', true);


--
-- Data for Name: app_custom_invoicings; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: customers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.customers (id, namespace, metadata, created_at, updated_at, deleted_at, name, currency, primary_email, billing_address_country, billing_address_postal_code, billing_address_state, billing_address_city, billing_address_line1, billing_address_line2, billing_address_phone_number, description, key) VALUES ('01JWB2ND14ZQ65N9N083RXTGZJ', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', NULL, '2025-05-28 09:13:06.85223+00', '2025-05-28 09:13:06.85223+00', NULL, 'Test Customer', 'USD', 'test@test.com', 'US', '12345', NULL, NULL, NULL, NULL, NULL, NULL, NULL);


--
-- Data for Name: app_custom_invoicing_customers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: app_customers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: app_stripes; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: app_stripe_customers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: entitlements; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: balance_snapshots; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_backup_migrated_flat_fees; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_customer_locks; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_customer_locks (id, namespace, customer_id) VALUES ('01JWB2ND393N8C2ZSB4BHZWCXS', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '01JWB2ND14ZQ65N9N083RXTGZJ');


--
-- Data for Name: billing_workflow_configs; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_workflow_configs (id, namespace, created_at, updated_at, deleted_at, collection_alignment, invoice_auto_advance, invoice_collection_method, line_collection_period, invoice_draft_period, invoice_due_after, invoice_progressive_billing, invoice_default_tax_settings, tax_enabled, tax_enforced) VALUES ('01JWB2ND0XSPE60CM2GEE0PK4C', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2025-05-28 09:13:06.84535+00', '2025-05-28 09:13:06.845351+00', NULL, 'subscription', true, 'charge_automatically', 'P0D', 'P1D', 'P1W', false, NULL, true, false);
INSERT INTO public.billing_workflow_configs (id, namespace, created_at, updated_at, deleted_at, collection_alignment, invoice_auto_advance, invoice_collection_method, line_collection_period, invoice_draft_period, invoice_due_after, invoice_progressive_billing, invoice_default_tax_settings, tax_enabled, tax_enforced) VALUES ('01JWB2ND40XHQTDXBBNZA6JQ0J', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2024-01-03 00:00:00+00', '2024-01-03 00:00:00+00', NULL, 'subscription', true, 'charge_automatically', 'P0D', 'P1D', 'P1W', false, NULL, true, false);


--
-- Data for Name: billing_profiles; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_profiles (id, namespace, created_at, updated_at, deleted_at, "default", workflow_config_id, metadata, supplier_address_country, supplier_address_postal_code, supplier_address_state, supplier_address_city, supplier_address_line1, supplier_address_line2, supplier_address_phone_number, supplier_name, name, description, supplier_tax_code, tax_app_id, invoicing_app_id, payment_app_id) VALUES ('01JWB2ND0YXXWNWYCPVNZQWB46', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2025-05-28 09:13:06.846356+00', '2025-05-28 09:13:06.846356+00', NULL, true, '01JWB2ND0XSPE60CM2GEE0PK4C', 'null', 'US', NULL, NULL, NULL, NULL, NULL, NULL, 'Awesome Supplier', 'Awesome Profile', NULL, NULL, '01JWB2ND0MDGNC6WZBBW8ZQCAC', '01JWB2ND0MDGNC6WZBBW8ZQCAC', '01JWB2ND0MDGNC6WZBBW8ZQCAC');


--
-- Data for Name: billing_customer_overrides; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_flat_fee_line_configs; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount, category, payment_term, index) VALUES ('01JWB2ND42F01QM0GVC37N6M1S', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 6, 'regular', 'in_advance', NULL);
INSERT INTO public.billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount, category, payment_term, index) VALUES ('01JWB2ND42F01QM0GVC3RY1SSC', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 6, 'regular', 'in_advance', NULL);
INSERT INTO public.billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount, category, payment_term, index) VALUES ('01JWB2ND42F01QM0GVC554XAE8', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 6, 'regular', 'in_advance', NULL);


--
-- Data for Name: billing_invoice_usage_based_line_configs; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoices; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoices (id, namespace, created_at, updated_at, deleted_at, metadata, customer_id, voided_at, currency, status, period_start, period_end, source_billing_profile_id, workflow_config_id, number, supplier_address_country, supplier_address_postal_code, supplier_address_state, supplier_address_city, supplier_address_line1, supplier_address_line2, supplier_address_phone_number, customer_address_country, customer_address_postal_code, customer_address_state, customer_address_city, customer_address_line1, customer_address_line2, customer_address_phone_number, supplier_name, supplier_tax_code, customer_name, type, description, issued_at, due_at, tax_app_id, invoicing_app_id, payment_app_id, draft_until, customer_usage_attribution, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, payment_app_external_id, collection_at, sent_to_customer_at, tax_app_external_id, status_details_cache, quantity_snapshoted_at) VALUES ('01JWB2ND40XHQTDXBBP18J9QJ0', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2024-01-03 00:00:00+00', '2024-01-03 00:00:00+00', NULL, 'null', '01JWB2ND14ZQ65N9N083RXTGZJ', NULL, 'USD', 'gathering', '2024-01-01 00:00:00+00', '2024-01-04 00:00:00+00', '01JWB2ND0YXXWNWYCPVNZQWB46', '01JWB2ND40XHQTDXBBNZA6JQ0J', 'GATHER-TECU-USD-1', 'US', NULL, NULL, NULL, NULL, NULL, NULL, 'US', '12345', NULL, NULL, NULL, NULL, NULL, 'Awesome Supplier', NULL, 'Test Customer', 'standard', NULL, NULL, NULL, '01JWB2ND0MDGNC6WZBBW8ZQCAC', '01JWB2ND0MDGNC6WZBBW8ZQCAC', '01JWB2ND0MDGNC6WZBBW8ZQCAC', NULL, '{"type": "customer_usage_attribution.v1", "subjectKeys": ["test"]}', 0, 0, 0, 0, 0, 0, 0, NULL, NULL, '2024-01-01 00:00:00+00', NULL, NULL, NULL, NULL);


--
-- Data for Name: plans; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.plans (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, key, version, currency, effective_from, effective_to, billables_must_align) VALUES ('01JWB2ND1T6MJ5KXTS0YQ9TD0Y', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'null', '2024-01-01 00:00:00+00', '2024-01-01 00:00:00+00', NULL, 'Test Plan', NULL, 'test-plan', 1, 'USD', NULL, NULL, false);


--
-- Data for Name: subscriptions; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.subscriptions (id, namespace, created_at, updated_at, deleted_at, metadata, active_from, active_to, currency, customer_id, name, description, plan_id, billables_must_align) VALUES ('01JWB2ND2A1JM4YN054JSSNGVP', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2024-01-01 00:00:00+00', '2024-01-01 00:00:00+00', NULL, 'null', '2024-01-01 00:00:00+00', NULL, 'USD', '01JWB2ND14ZQ65N9N083RXTGZJ', 'subs-1', NULL, '01JWB2ND1T6MJ5KXTS0YQ9TD0Y', false);


--
-- Data for Name: subscription_phases; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.subscription_phases (id, namespace, created_at, updated_at, deleted_at, metadata, key, name, description, active_from, subscription_id) VALUES ('01JWB2ND2C2JHF9ZNA8QFATT74', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2024-01-01 00:00:00+00', '2024-01-01 00:00:00+00', NULL, 'null', 'first-phase', 'first-phase', NULL, '2024-01-01 00:00:00+00', '01JWB2ND2A1JM4YN054JSSNGVP');


--
-- Data for Name: subscription_items; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.subscription_items (id, namespace, created_at, updated_at, deleted_at, metadata, active_from, active_to, key, active_from_override_relative_to_phase_start, active_to_override_relative_to_phase_start, name, description, feature_key, entitlement_template, tax_config, billing_cadence, price, entitlement_id, phase_id, restarts_billing_period, discounts, annotations) VALUES ('01JWB2ND2EG2B716S8W6NKRRJK', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', '2024-01-01 00:00:00+00', '2024-01-01 00:00:00+00', NULL, NULL, '2024-01-01 00:00:00+00', NULL, 'in-advance', NULL, NULL, 'in-advance', NULL, NULL, NULL, NULL, 'P1D', '{"type": "flat", "amount": "6", "paymentTerm": "in_advance"}', NULL, '01JWB2ND2C2JHF9ZNA8QFATT74', NULL, NULL, '{"subscription.owner": ["subscription"]}');


--
-- Data for Name: billing_invoice_lines; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts) VALUES ('01JWB2ND43KPCHMXHKD0KETF5M', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'null', '2024-01-03 00:00:00+00', '2024-01-03 00:00:00+00', NULL, 'in-advance', NULL, '2024-01-03 00:00:00+00', '2024-01-04 00:00:00+00', '2024-01-03 00:00:00+00', 'flat_fee', 'valid', 'USD', 1, NULL, '01JWB2ND40XHQTDXBBP18J9QJ0', '01JWB2ND42F01QM0GVC554XAE8', NULL, NULL, '01JWB2ND2A1JM4YN054JSSNGVP/first-phase/in-advance/v[0]/period[2]', 6, 0, 0, 0, 0, 0, 6, NULL, '01JWB2ND2A1JM4YN054JSSNGVP', '01JWB2ND2EG2B716S8W6NKRRJK', '01JWB2ND2C2JHF9ZNA8QFATT74', NULL, 'subscription', NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts) VALUES ('01JWB2ND43KPCHMXHKCW3PFW6W', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'null', '2024-01-03 00:00:00+00', '2024-01-03 00:00:00+00', '2024-01-03 00:00:00+00', 'in-advance', NULL, '2024-01-01 00:00:00+00', '2024-01-02 00:00:00+00', '2024-01-01 00:00:00+00', 'flat_fee', 'valid', 'USD', 1, NULL, '01JWB2ND40XHQTDXBBP18J9QJ0', '01JWB2ND42F01QM0GVC37N6M1S', NULL, NULL, '01JWB2ND2A1JM4YN054JSSNGVP/first-phase/in-advance/v[0]/period[0]', 6, 0, 0, 0, 0, 0, 6, NULL, '01JWB2ND2A1JM4YN054JSSNGVP', '01JWB2ND2EG2B716S8W6NKRRJK', '01JWB2ND2C2JHF9ZNA8QFATT74', NULL, 'subscription', NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts) VALUES ('01JWB2ND43KPCHMXHKCYEQ13WY', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'null', '2024-01-03 00:00:00+00', '2024-01-03 00:00:00+00', NULL, 'in-advance', NULL, '2024-01-02 00:00:00+00', '2024-01-03 00:00:00+00', '2024-01-02 00:00:00+00', 'flat_fee', 'valid', 'USD', 1, NULL, '01JWB2ND40XHQTDXBBP18J9QJ0', '01JWB2ND42F01QM0GVC3RY1SSC', NULL, NULL, '01JWB2ND2A1JM4YN054JSSNGVP/first-phase/in-advance/v[0]/period[1]', 6, 0, 0, 0, 0, 0, 6, 'invoicing-external-id', '01JWB2ND2A1JM4YN054JSSNGVP', '01JWB2ND2EG2B716S8W6NKRRJK', '01JWB2ND2C2JHF9ZNA8QFATT74', NULL, 'subscription', NULL);


--
-- Data for Name: billing_invoice_line_discounts; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_line_usage_discounts; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_validation_issues; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_sequence_numbers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_sequence_numbers (id, namespace, scope, last) VALUES (1, 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'invoices/gathering', 1);


--
-- Data for Name: customer_subjects; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.customer_subjects (id, subject_key, created_at, customer_id, namespace, deleted_at) VALUES (1, 'test', '2025-05-28 09:13:06.856315+00', '01JWB2ND14ZQ65N9N083RXTGZJ', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', NULL);


--
-- Data for Name: grants; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: meters; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: notification_channels; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: notification_rules; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: notification_channel_rules; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: notification_event_delivery_status; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: notification_events; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: notification_event_delivery_status_events; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: plan_addons; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: plan_phases; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.plan_phases (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, key, plan_id, index, duration) VALUES ('01JWB2ND1WGAXGZZ31XMHGQNF0', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'null', '2024-01-01 00:00:00+00', '2024-01-01 00:00:00+00', NULL, 'first-phase', NULL, 'first-phase', '01JWB2ND1T6MJ5KXTS0YQ9TD0Y', 0, NULL);


--
-- Data for Name: plan_rate_cards; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.plan_rate_cards (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, key, type, feature_key, entitlement_template, tax_config, billing_cadence, price, feature_id, phase_id, discounts) VALUES ('01JWB2ND1YJT84V9PRV4QVHFVT', 'test-subs-update-01JWB2ND0EBJWE1FX8EN9AVVJP', 'null', '2024-01-01 00:00:00+00', '2024-01-01 00:00:00+00', NULL, 'in-advance', NULL, 'in-advance', 'usage_based', NULL, 'null', NULL, 'P1D', '{"type": "flat", "amount": "6", "paymentTerm": "in_advance"}', NULL, '01JWB2ND1WGAXGZZ31XMHGQNF0', 'null');


--
-- Data for Name: subscription_addons; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: subscription_addon_quantities; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: usage_resets; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Name: app_custom_invoicing_customers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.app_custom_invoicing_customers_id_seq', 1, false);


--
-- Name: app_customers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.app_customers_id_seq', 1, false);


--
-- Name: app_stripe_customers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.app_stripe_customers_id_seq', 1, false);


--
-- Name: balance_snapshots_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.balance_snapshots_id_seq', 1, false);


--
-- Name: billing_sequence_numbers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.billing_sequence_numbers_id_seq', 1, true);


--
-- Name: customer_subjects_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.customer_subjects_id_seq', 1, true);


--
-- PostgreSQL database dump complete
--

