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

INSERT INTO public.features (id, created_at, updated_at, deleted_at, metadata, namespace, name, key, meter_slug, meter_group_by_filters, archived_at) VALUES ('01JXA7Y5BEX8SBN22BT367VVZ4', '2025-06-09 11:41:44.174624+00', '2025-06-09 11:41:44.174624+00', NULL, 'null', 'ns-ubp-invoicing-progressive', 'flat-per-unit', 'flat-per-unit', 'flat-per-unit', NULL, NULL);


--
-- Data for Name: addon_rate_cards; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: apps; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.apps (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, type, status, is_default) VALUES ('01JXA7Y5BBM0M8D504ZNQ6WX8X', 'ns-ubp-invoicing-progressive', NULL, '2025-06-09 11:41:44.171041+00', '2025-06-09 11:41:44.171042+00', NULL, 'Sandbox', 'Sandbox app', 'sandbox', 'ready', true);


--
-- Data for Name: app_custom_invoicings; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: customers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.customers (id, namespace, metadata, created_at, updated_at, deleted_at, name, currency, primary_email, billing_address_country, billing_address_postal_code, billing_address_state, billing_address_city, billing_address_line1, billing_address_line2, billing_address_phone_number, description, key) VALUES ('01JXA7Y5BFE60CQ5EJZ1QYQQ5G', 'ns-ubp-invoicing-progressive', NULL, '2025-06-09 11:41:44.17593+00', '2025-06-09 11:41:44.17593+00', NULL, 'Test Customer', 'USD', 'test@test.com', 'US', '12345', NULL, NULL, NULL, NULL, NULL, NULL, NULL);


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
-- Data for Name: billing_customer_locks; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_customer_locks (id, namespace, customer_id) VALUES ('01JXA7Y5BZH1DTFFFWR8GDPE23', 'ns-ubp-invoicing-progressive', '01JXA7Y5BFE60CQ5EJZ1QYQQ5G');


--
-- Data for Name: billing_workflow_configs; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_workflow_configs (id, namespace, created_at, updated_at, deleted_at, collection_alignment, invoice_auto_advance, invoice_collection_method, line_collection_period, invoice_draft_period, invoice_due_after, invoice_progressive_billing, invoice_default_tax_settings, tax_enabled, tax_enforced) VALUES ('01JXA7Y5BSZPMR8ZKM42H6HENZ', 'ns-ubp-invoicing-progressive', '2025-06-09 11:41:44.185068+00', '2025-06-09 11:41:44.185068+00', NULL, 'subscription', true, 'charge_automatically', 'P0D', 'P1D', 'P1W', true, NULL, true, false);
INSERT INTO public.billing_workflow_configs (id, namespace, created_at, updated_at, deleted_at, collection_alignment, invoice_auto_advance, invoice_collection_method, line_collection_period, invoice_draft_period, invoice_due_after, invoice_progressive_billing, invoice_default_tax_settings, tax_enabled, tax_enforced) VALUES ('01JXA7Y5CM63E7618SFEJPTQ3S', 'ns-ubp-invoicing-progressive', '2025-06-09 11:41:44.212724+00', '2024-09-02 13:13:14.067284+00', NULL, 'subscription', true, 'charge_automatically', 'P0D', 'P1D', 'P1W', true, NULL, true, false);
INSERT INTO public.billing_workflow_configs (id, namespace, created_at, updated_at, deleted_at, collection_alignment, invoice_auto_advance, invoice_collection_method, line_collection_period, invoice_draft_period, invoice_due_after, invoice_progressive_billing, invoice_default_tax_settings, tax_enabled, tax_enforced) VALUES ('01JXA7Y5DTY3GPA1VNMP84JBJB', 'ns-ubp-invoicing-progressive', '2024-09-02 13:13:14.010524+00', '2024-09-02 13:13:14.095593+00', NULL, 'subscription', true, 'charge_automatically', 'P0D', 'P1D', 'P1W', true, NULL, true, false);


--
-- Data for Name: billing_profiles; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_profiles (id, namespace, created_at, updated_at, deleted_at, "default", workflow_config_id, metadata, supplier_address_country, supplier_address_postal_code, supplier_address_state, supplier_address_city, supplier_address_line1, supplier_address_line2, supplier_address_phone_number, supplier_name, name, description, supplier_tax_code, tax_app_id, invoicing_app_id, payment_app_id) VALUES ('01JXA7Y5BSZPMR8ZKM45C10E6T', 'ns-ubp-invoicing-progressive', '2025-06-09 11:41:44.185994+00', '2025-06-09 11:41:44.185995+00', NULL, true, '01JXA7Y5BSZPMR8ZKM42H6HENZ', 'null', 'US', NULL, NULL, NULL, NULL, NULL, NULL, 'Awesome Supplier', 'Awesome Profile', NULL, NULL, '01JXA7Y5BBM0M8D504ZNQ6WX8X', '01JXA7Y5BBM0M8D504ZNQ6WX8X', '01JXA7Y5BBM0M8D504ZNQ6WX8X');


--
-- Data for Name: billing_customer_overrides; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_flat_fee_line_configs; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount, category, payment_term, index) VALUES ('01JXA7Y5ESSNK0A1D77JHWBN61', 'ns-ubp-invoicing-progressive', 100, 'regular', 'in_arrears', 0);
INSERT INTO public.billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount, category, payment_term, index) VALUES ('01JXA7Y5FKC57VN3BGZK2VB5B2', 'ns-ubp-invoicing-progressive', 100, 'regular', 'in_arrears', 0);
INSERT INTO public.billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount, category, payment_term, index) VALUES ('01JXA7Y5GFYA1V39TS5HGXQEC6', 'ns-ubp-invoicing-progressive', 100, 'regular', 'in_arrears', 0);


--
-- Data for Name: plans; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: subscriptions; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: subscription_phases; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: subscription_items; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_split_line_groups; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_usage_based_line_configs; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoice_usage_based_line_configs (id, namespace, price_type, feature_key, price, pre_line_period_quantity, metered_quantity, metered_pre_line_period_quantity) VALUES ('01JXA7Y5CQ74JPSBM8FZD65QM1', 'ns-ubp-invoicing-progressive', 'unit', 'flat-per-unit', '{"type": "unit", "amount": "100", "maximumAmount": "2000"}', NULL, NULL, NULL);
INSERT INTO public.billing_invoice_usage_based_line_configs (id, namespace, price_type, feature_key, price, pre_line_period_quantity, metered_quantity, metered_pre_line_period_quantity) VALUES ('01JXA7Y5CQ74JPSBM8G1JCA42B', 'ns-ubp-invoicing-progressive', 'flat', '', '{"type": "flat", "amount": "100", "paymentTerm": "in_arrears"}', NULL, NULL, NULL);
INSERT INTO public.billing_invoice_usage_based_line_configs (id, namespace, price_type, feature_key, price, pre_line_period_quantity, metered_quantity, metered_pre_line_period_quantity) VALUES ('01JXA7Y5E765KJ1F43EV0TV3YV', 'ns-ubp-invoicing-progressive', 'unit', 'flat-per-unit', '{"type": "unit", "amount": "100", "maximumAmount": "2000"}', NULL, NULL, NULL);
INSERT INTO public.billing_invoice_usage_based_line_configs (id, namespace, price_type, feature_key, price, pre_line_period_quantity, metered_quantity, metered_pre_line_period_quantity) VALUES ('01JXA7Y5E04HGZT2CHR62S0W72', 'ns-ubp-invoicing-progressive', 'unit', 'flat-per-unit', '{"type": "unit", "amount": "100", "maximumAmount": "2000"}', 0, 10, 0);


--
-- Data for Name: billing_invoices; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoices (id, namespace, created_at, updated_at, deleted_at, metadata, customer_id, voided_at, currency, status, period_start, period_end, source_billing_profile_id, workflow_config_id, number, supplier_address_country, supplier_address_postal_code, supplier_address_state, supplier_address_city, supplier_address_line1, supplier_address_line2, supplier_address_phone_number, customer_address_country, customer_address_postal_code, customer_address_state, customer_address_city, customer_address_line1, customer_address_line2, customer_address_phone_number, supplier_name, supplier_tax_code, customer_name, type, description, issued_at, due_at, tax_app_id, invoicing_app_id, payment_app_id, draft_until, customer_usage_attribution, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, payment_app_external_id, collection_at, sent_to_customer_at, tax_app_external_id, status_details_cache, quantity_snapshoted_at) VALUES ('01JXA7Y5CN0DDF3J98X77PT09W', 'ns-ubp-invoicing-progressive', '2025-06-09 11:41:44.213161+00', '2024-09-02 13:13:14.064335+00', NULL, 'null', '01JXA7Y5BFE60CQ5EJZ1QYQQ5G', NULL, 'USD', 'gathering', '2024-09-02 12:13:14+00', '2024-09-03 12:13:14+00', '01JXA7Y5BSZPMR8ZKM45C10E6T', '01JXA7Y5CM63E7618SFEJPTQ3S', 'GATHER-TECU-USD-1', 'US', NULL, NULL, NULL, NULL, NULL, NULL, 'US', '12345', NULL, NULL, NULL, NULL, NULL, 'Awesome Supplier', NULL, 'Test Customer', 'standard', NULL, NULL, NULL, '01JXA7Y5BBM0M8D504ZNQ6WX8X', '01JXA7Y5BBM0M8D504ZNQ6WX8X', '01JXA7Y5BBM0M8D504ZNQ6WX8X', '2025-06-10 11:41:44.213161+00', '{"type": "customer_usage_attribution.v1", "subjectKeys": ["test"]}', 100, 0, 0, 0, 0, 0, 100, NULL, NULL, '2025-06-09 11:41:44.213161+00', NULL, NULL, NULL, NULL);
INSERT INTO public.billing_invoices (id, namespace, created_at, updated_at, deleted_at, metadata, customer_id, voided_at, currency, status, period_start, period_end, source_billing_profile_id, workflow_config_id, number, supplier_address_country, supplier_address_postal_code, supplier_address_state, supplier_address_city, supplier_address_line1, supplier_address_line2, supplier_address_phone_number, customer_address_country, customer_address_postal_code, customer_address_state, customer_address_city, customer_address_line1, customer_address_line2, customer_address_phone_number, supplier_name, supplier_tax_code, customer_name, type, description, issued_at, due_at, tax_app_id, invoicing_app_id, payment_app_id, draft_until, customer_usage_attribution, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, payment_app_external_id, collection_at, sent_to_customer_at, tax_app_external_id, status_details_cache, quantity_snapshoted_at) VALUES ('01JXA7Y5DTY3GPA1VNMPD649TK', 'ns-ubp-invoicing-progressive', '2024-09-02 13:13:14.010891+00', '2024-09-02 13:13:14.093814+00', NULL, 'null', '01JXA7Y5BFE60CQ5EJZ1QYQQ5G', NULL, 'USD', 'draft.waiting_auto_approval', '2024-09-02 12:13:00+00', '2024-09-02 13:13:00+00', '01JXA7Y5BSZPMR8ZKM45C10E6T', '01JXA7Y5DTY3GPA1VNMP84JBJB', 'DRAFT-AWPR-1', 'US', NULL, NULL, NULL, NULL, NULL, NULL, 'US', '12345', NULL, NULL, NULL, NULL, NULL, 'Awesome Supplier', NULL, 'Test Customer', 'standard', NULL, NULL, '2024-09-09 13:13:14+00', '01JXA7Y5BBM0M8D504ZNQ6WX8X', '01JXA7Y5BBM0M8D504ZNQ6WX8X', '01JXA7Y5BBM0M8D504ZNQ6WX8X', '2024-09-03 13:13:14.010891+00', '{"type": "customer_usage_attribution.v1", "subjectKeys": ["test"]}', 1000, 0, 0, 0, 0, 0, 1000, NULL, NULL, '2024-09-02 13:13:14.010891+00', NULL, NULL, '{"failed": false, "immutable": false, "availableActions": {"delete": {"resultingState": "deleted"}, "approve": {"resultingState": "payment_processing.pending"}}}', '2024-09-02 13:13:14.091153+00');


--
-- Data for Name: billing_invoice_lines; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts, split_line_group_id) VALUES ('01JXA7Y5CRJF0NJ5ADKNZVDTGH', 'ns-ubp-invoicing-progressive', 'null', '2025-06-09 11:41:44.216948+00', '2024-09-02 13:13:14.045608+00', NULL, 'UBP - FLAT per unit', NULL, '2024-09-02 12:13:00+00', '2024-09-03 12:13:00+00', '2024-09-03 12:13:14+00', 'usage_based', 'split', 'USD', NULL, NULL, '01JXA7Y5CN0DDF3J98X77PT09W', NULL, '01JXA7Y5CQ74JPSBM8FZD65QM1', NULL, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL, NULL, NULL, NULL, 'manual', NULL, NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts, split_line_group_id) VALUES ('01JXA7Y5E765KJ1F43EXGHNJTN', 'ns-ubp-invoicing-progressive', 'null', '2024-09-02 13:13:14.024368+00', '2024-09-02 13:13:14.024369+00', NULL, 'UBP - FLAT per unit', NULL, '2024-09-02 13:13:00+00', '2024-09-03 12:13:00+00', '2024-09-03 12:13:14+00', 'usage_based', 'valid', 'USD', NULL, NULL, '01JXA7Y5CN0DDF3J98X77PT09W', NULL, '01JXA7Y5E765KJ1F43EV0TV3YV', '01JXA7Y5CRJF0NJ5ADKNZVDTGH', NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL, NULL, NULL, NULL, 'manual', NULL, NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts, split_line_group_id) VALUES ('01JXA7Y5CRJF0NJ5ADKNZYNCVD', 'ns-ubp-invoicing-progressive', 'null', '2025-06-09 11:41:44.216948+00', '2024-09-02 13:13:14.068374+00', NULL, 'UBP - FLAT per any usage', NULL, '2024-09-02 12:13:14+00', '2024-09-03 12:13:14+00', '2024-09-03 12:13:14+00', 'usage_based', 'valid', 'USD', 1, NULL, '01JXA7Y5CN0DDF3J98X77PT09W', NULL, '01JXA7Y5CQ74JPSBM8G1JCA42B', NULL, NULL, 100, 0, 0, 0, 0, 0, 100, NULL, NULL, NULL, NULL, NULL, 'manual', NULL, NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts, split_line_group_id) VALUES ('01JXA7Y5E18XC1ZTKZ59B342CM', 'ns-ubp-invoicing-progressive', 'null', '2024-09-02 13:13:14.017513+00', '2024-09-02 13:13:14.097227+00', NULL, 'UBP - FLAT per unit', NULL, '2024-09-02 12:13:00+00', '2024-09-02 13:13:00+00', '2024-09-02 13:13:00+00', 'usage_based', 'valid', 'USD', 10, NULL, '01JXA7Y5DTY3GPA1VNMPD649TK', NULL, '01JXA7Y5E04HGZT2CHR62S0W72', '01JXA7Y5CRJF0NJ5ADKNZVDTGH', NULL, 1000, 0, 0, 0, 0, 0, 1000, NULL, NULL, NULL, NULL, NULL, 'manual', NULL, NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts, split_line_group_id) VALUES ('01JXA7Y5FMER9E0ZJMFEJ830S4', 'ns-ubp-invoicing-progressive', 'null', '2024-09-02 13:13:14.069163+00', '2024-09-02 13:13:14.069163+00', NULL, 'UBP - FLAT per any usage', NULL, '2024-09-02 12:13:14+00', '2024-09-03 12:13:14+00', '2024-09-03 12:13:14+00', 'flat_fee', 'detailed', 'USD', 1, NULL, '01JXA7Y5CN0DDF3J98X77PT09W', '01JXA7Y5FKC57VN3BGZK2VB5B2', NULL, '01JXA7Y5CRJF0NJ5ADKNZYNCVD', 'flat-price', 100, 0, 0, 0, 0, 0, 100, NULL, NULL, NULL, NULL, NULL, 'system', NULL, NULL);
INSERT INTO public.billing_invoice_lines (id, namespace, metadata, created_at, updated_at, deleted_at, name, description, period_start, period_end, invoice_at, type, status, currency, quantity, tax_config, invoice_id, fee_line_config_id, usage_based_line_config_id, parent_line_id, child_unique_reference_id, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total, total, invoicing_app_external_id, subscription_id, subscription_item_id, subscription_phase_id, line_ids, managed_by, ratecard_discounts, split_line_group_id) VALUES ('01JXA7Y5EWQ5MGXD2W3K8SWBXX', 'ns-ubp-invoicing-progressive', 'null', '2024-09-02 13:13:14.044657+00', '2024-09-02 13:13:14.096672+00', NULL, 'UBP - FLAT per unit: usage in period', NULL, '2024-09-02 12:13:00+00', '2024-09-02 13:13:00+00', '2024-09-02 13:13:00+00', 'flat_fee', 'detailed', 'USD', 10, NULL, '01JXA7Y5DTY3GPA1VNMPD649TK', '01JXA7Y5GFYA1V39TS5HGXQEC6', NULL, '01JXA7Y5E18XC1ZTKZ59B342CM', 'unit-price-usage', 1000, 0, 0, 0, 0, 0, 1000, NULL, NULL, NULL, NULL, NULL, 'system', NULL, NULL);

--
-- Data for Name: billing_invoice_line_discounts; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_line_usage_discounts; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: billing_invoice_validation_issues; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_invoice_validation_issues (id, namespace, created_at, updated_at, deleted_at, severity, code, message, path, component, dedupe_hash, invoice_id) VALUES ('01JXA7Y5FHPVB3R6DRPHJV60KS', 'ns-ubp-invoicing-progressive', '2024-09-02 13:13:14.066307+00', '2024-09-02 13:13:14.066308+00', NULL, 'critical', NULL, 'quantity and pre-line period quantity must be set for line[01JXA7Y5E765KJ1F43EXGHNJTN]', '/line[01JXA7Y5E765KJ1F43EXGHNJTN]', 'openmeter', '\x20d9d1af5a95e36c10d436736922e25c3d36b78dbdd5f6f86fa04ab9ddf860d7', '01JXA7Y5CN0DDF3J98X77PT09W');


--
-- Data for Name: billing_sequence_numbers; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.billing_sequence_numbers (id, namespace, scope, last) VALUES (1, 'ns-ubp-invoicing-progressive', 'invoices/gathering', 1);
INSERT INTO public.billing_sequence_numbers (id, namespace, scope, last) VALUES (2, 'ns-ubp-invoicing-progressive', 'invoices/draft', 1);


--
-- Data for Name: customer_subjects; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--

INSERT INTO public.customer_subjects (id, subject_key, created_at, customer_id, namespace, deleted_at) VALUES (1, 'test', '2025-06-09 11:41:44.176953+00', '01JXA7Y5BFE60CQ5EJZ1QYQQ5G', 'ns-ubp-invoicing-progressive', NULL);


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



--
-- Data for Name: plan_rate_cards; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



--
-- Data for Name: subjects; Type: TABLE DATA; Schema: public; Owner: pgtdbuser
--



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

SELECT pg_catalog.setval('public.billing_sequence_numbers_id_seq', 2, true);


--
-- Name: customer_subjects_id_seq; Type: SEQUENCE SET; Schema: public; Owner: pgtdbuser
--

SELECT pg_catalog.setval('public.customer_subjects_id_seq', 1, true);


--
-- PostgreSQL database dump complete
--

