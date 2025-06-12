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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: billing_customer_locks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_customer_locks (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    customer_id character(26) NOT NULL
);


--
-- Name: billing_customer_overrides; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_customer_overrides (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    collection_alignment character varying,
    invoice_auto_advance boolean,
    invoice_collection_method character varying,
    billing_profile_id character(26),
    customer_id character(26) NOT NULL,
    line_collection_period character varying,
    invoice_draft_period character varying,
    invoice_due_after character varying,
    invoice_progressive_billing boolean,
    invoice_default_tax_config jsonb
);


--
-- Name: billing_invoice_flat_fee_line_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_flat_fee_line_configs (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    per_unit_amount numeric NOT NULL,
    category character varying DEFAULT 'regular'::character varying NOT NULL,
    payment_term character varying DEFAULT 'in_advance'::character varying NOT NULL,
    index bigint
);


--
-- Name: billing_invoice_line_discounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_line_discounts (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    child_unique_reference_id character varying,
    description character varying,
    amount numeric NOT NULL,
    line_id character(26) NOT NULL,
    invoicing_app_external_id character varying,
    reason character varying NOT NULL,
    type character varying,
    rounding_amount numeric,
    quantity numeric,
    pre_line_period_quantity numeric,
    source_discount jsonb
);


--
-- Name: billing_invoice_line_usage_discounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_line_usage_discounts (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    child_unique_reference_id character varying,
    description character varying,
    reason character varying NOT NULL,
    invoicing_app_external_id character varying,
    quantity numeric NOT NULL,
    pre_line_period_quantity numeric,
    reason_details jsonb,
    line_id character(26) NOT NULL
);


--
-- Name: billing_invoice_lines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_lines (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    metadata jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    name character varying NOT NULL,
    description character varying,
    period_start timestamp with time zone NOT NULL,
    period_end timestamp with time zone NOT NULL,
    invoice_at timestamp with time zone NOT NULL,
    type character varying NOT NULL,
    status character varying NOT NULL,
    currency character varying(3) NOT NULL,
    quantity numeric,
    tax_config jsonb,
    invoice_id character(26) NOT NULL,
    fee_line_config_id character(26),
    usage_based_line_config_id character(26),
    parent_line_id character(26),
    child_unique_reference_id character varying,
    amount numeric NOT NULL,
    taxes_total numeric NOT NULL,
    taxes_inclusive_total numeric NOT NULL,
    taxes_exclusive_total numeric NOT NULL,
    charges_total numeric NOT NULL,
    discounts_total numeric NOT NULL,
    total numeric NOT NULL,
    invoicing_app_external_id character varying,
    subscription_id character(26),
    subscription_item_id character(26),
    subscription_phase_id character(26),
    line_ids character(26),
    managed_by character varying NOT NULL,
    ratecard_discounts jsonb,
    split_line_group_id character(26)
);


--
-- Name: billing_invoice_split_line_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_split_line_groups (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    metadata jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    name character varying NOT NULL,
    description character varying,
    service_period_start timestamp with time zone NOT NULL,
    service_period_end timestamp with time zone NOT NULL,
    currency character varying(3) NOT NULL,
    tax_config jsonb,
    unique_reference_id character varying,
    ratecard_discounts jsonb,
    feature_key character varying,
    price jsonb NOT NULL,
    subscription_id character(26),
    subscription_item_id character(26),
    subscription_phase_id character(26)
);


--
-- Name: billing_invoice_usage_based_line_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_usage_based_line_configs (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    price_type character varying NOT NULL,
    feature_key character varying,
    price jsonb NOT NULL,
    pre_line_period_quantity numeric,
    metered_quantity numeric,
    metered_pre_line_period_quantity numeric
);


--
-- Name: billing_invoice_validation_issues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoice_validation_issues (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    severity character varying NOT NULL,
    code character varying,
    message character varying NOT NULL,
    path character varying,
    component character varying NOT NULL,
    dedupe_hash bytea NOT NULL,
    invoice_id character(26) NOT NULL
);


--
-- Name: billing_invoices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_invoices (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    metadata jsonb,
    customer_id character(26) NOT NULL,
    voided_at timestamp with time zone,
    currency character varying(3) NOT NULL,
    status character varying NOT NULL,
    period_start timestamp with time zone,
    period_end timestamp with time zone,
    source_billing_profile_id character(26) NOT NULL,
    workflow_config_id character(26) NOT NULL,
    number character varying NOT NULL,
    supplier_address_country character varying,
    supplier_address_postal_code character varying,
    supplier_address_state character varying,
    supplier_address_city character varying,
    supplier_address_line1 character varying,
    supplier_address_line2 character varying,
    supplier_address_phone_number character varying,
    customer_address_country character varying,
    customer_address_postal_code character varying,
    customer_address_state character varying,
    customer_address_city character varying,
    customer_address_line1 character varying,
    customer_address_line2 character varying,
    customer_address_phone_number character varying,
    supplier_name character varying NOT NULL,
    supplier_tax_code character varying,
    customer_name character varying NOT NULL,
    type character varying NOT NULL,
    description character varying,
    issued_at timestamp with time zone,
    due_at timestamp with time zone,
    tax_app_id character(26) NOT NULL,
    invoicing_app_id character(26) NOT NULL,
    payment_app_id character(26) NOT NULL,
    draft_until timestamp with time zone,
    customer_usage_attribution jsonb NOT NULL,
    amount numeric NOT NULL,
    taxes_total numeric NOT NULL,
    taxes_inclusive_total numeric NOT NULL,
    taxes_exclusive_total numeric NOT NULL,
    charges_total numeric NOT NULL,
    discounts_total numeric NOT NULL,
    total numeric NOT NULL,
    invoicing_app_external_id character varying,
    payment_app_external_id character varying,
    collection_at timestamp with time zone,
    sent_to_customer_at timestamp with time zone,
    tax_app_external_id character varying,
    status_details_cache jsonb,
    quantity_snapshoted_at timestamp with time zone
);


--
-- Name: billing_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_profiles (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    "default" boolean DEFAULT false NOT NULL,
    workflow_config_id character(26) NOT NULL,
    metadata jsonb,
    supplier_address_country character varying,
    supplier_address_postal_code character varying,
    supplier_address_state character varying,
    supplier_address_city character varying,
    supplier_address_line1 character varying,
    supplier_address_line2 character varying,
    supplier_address_phone_number character varying,
    supplier_name character varying NOT NULL,
    name character varying NOT NULL,
    description character varying,
    supplier_tax_code character varying,
    tax_app_id character(26) NOT NULL,
    invoicing_app_id character(26) NOT NULL,
    payment_app_id character(26) NOT NULL
);


--
-- Name: billing_sequence_numbers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_sequence_numbers (
    id bigint NOT NULL,
    namespace character varying NOT NULL,
    scope character varying NOT NULL,
    last numeric NOT NULL
);


--
-- Name: billing_sequence_numbers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.billing_sequence_numbers ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.billing_sequence_numbers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: billing_workflow_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_workflow_configs (
    id character(26) NOT NULL,
    namespace character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    collection_alignment character varying NOT NULL,
    invoice_auto_advance boolean NOT NULL,
    invoice_collection_method character varying NOT NULL,
    line_collection_period character varying NOT NULL,
    invoice_draft_period character varying NOT NULL,
    invoice_due_after character varying NOT NULL,
    invoice_progressive_billing boolean NOT NULL,
    invoice_default_tax_settings jsonb,
    tax_enabled boolean DEFAULT true NOT NULL,
    tax_enforced boolean DEFAULT false NOT NULL
);


--
-- Name: billing_customer_locks billing_customer_locks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_customer_locks
    ADD CONSTRAINT billing_customer_locks_pkey PRIMARY KEY (id);


--
-- Name: billing_customer_overrides billing_customer_overrides_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_customer_overrides
    ADD CONSTRAINT billing_customer_overrides_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_flat_fee_line_configs billing_invoice_flat_fee_line_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_flat_fee_line_configs
    ADD CONSTRAINT billing_invoice_flat_fee_line_configs_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_line_discounts billing_invoice_line_discounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_line_discounts
    ADD CONSTRAINT billing_invoice_line_discounts_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_line_usage_discounts billing_invoice_line_usage_discounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_line_usage_discounts
    ADD CONSTRAINT billing_invoice_line_usage_discounts_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_lines billing_invoice_lines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_split_line_groups billing_invoice_split_line_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_split_line_groups
    ADD CONSTRAINT billing_invoice_split_line_groups_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_usage_based_line_configs billing_invoice_usage_based_line_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_usage_based_line_configs
    ADD CONSTRAINT billing_invoice_usage_based_line_configs_pkey PRIMARY KEY (id);


--
-- Name: billing_invoice_validation_issues billing_invoice_validation_issues_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_validation_issues
    ADD CONSTRAINT billing_invoice_validation_issues_pkey PRIMARY KEY (id);


--
-- Name: billing_invoices billing_invoices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_pkey PRIMARY KEY (id);


--
-- Name: billing_profiles billing_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_profiles
    ADD CONSTRAINT billing_profiles_pkey PRIMARY KEY (id);


--
-- Name: billing_sequence_numbers billing_sequence_numbers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_sequence_numbers
    ADD CONSTRAINT billing_sequence_numbers_pkey PRIMARY KEY (id);


--
-- Name: billing_workflow_configs billing_workflow_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_workflow_configs
    ADD CONSTRAINT billing_workflow_configs_pkey PRIMARY KEY (id);


--
-- Name: billing_customer_overrides_customer_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billing_customer_overrides_customer_id_key ON public.billing_customer_overrides USING btree (customer_id);


--
-- Name: billing_invoices_workflow_config_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billing_invoices_workflow_config_id_key ON public.billing_invoices USING btree (workflow_config_id);


--
-- Name: billing_profiles_workflow_config_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billing_profiles_workflow_config_id_key ON public.billing_profiles USING btree (workflow_config_id);


--
-- Name: billingcustomerlock_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingcustomerlock_id ON public.billing_customer_locks USING btree (id);


--
-- Name: billingcustomerlock_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billingcustomerlock_namespace ON public.billing_customer_locks USING btree (namespace);


--
-- Name: billingcustomerlock_namespace_customer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingcustomerlock_namespace_customer_id ON public.billing_customer_locks USING btree (namespace, customer_id);


--
-- Name: billingcustomeroverride_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingcustomeroverride_id ON public.billing_customer_overrides USING btree (id);


--
-- Name: billingcustomeroverride_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billingcustomeroverride_namespace ON public.billing_customer_overrides USING btree (namespace);


--
-- Name: billingcustomeroverride_namespace_customer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingcustomeroverride_namespace_customer_id ON public.billing_customer_overrides USING btree (namespace, customer_id);


--
-- Name: billingcustomeroverride_namespace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingcustomeroverride_namespace_id ON public.billing_customer_overrides USING btree (namespace, id);


--
-- Name: billinginvoice_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoice_id ON public.billing_invoices USING btree (id);


--
-- Name: billinginvoice_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoice_namespace ON public.billing_invoices USING btree (namespace);


--
-- Name: billinginvoice_namespace_customer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoice_namespace_customer_id ON public.billing_invoices USING btree (namespace, customer_id);


--
-- Name: billinginvoice_namespace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoice_namespace_id ON public.billing_invoices USING btree (namespace, id);


--
-- Name: billinginvoice_namespace_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoice_namespace_status ON public.billing_invoices USING btree (namespace, status);


--
-- Name: billinginvoice_status_details_cache; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoice_status_details_cache ON public.billing_invoices USING gin (status_details_cache);


--
-- Name: billinginvoiceflatfeelineconfig_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoiceflatfeelineconfig_id ON public.billing_invoice_flat_fee_line_configs USING btree (id);


--
-- Name: billinginvoiceflatfeelineconfig_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoiceflatfeelineconfig_namespace ON public.billing_invoice_flat_fee_line_configs USING btree (namespace);


--
-- Name: billinginvoiceline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoiceline_id ON public.billing_invoice_lines USING btree (id);


--
-- Name: billinginvoiceline_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoiceline_namespace ON public.billing_invoice_lines USING btree (namespace);


--
-- Name: billinginvoiceline_namespace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoiceline_namespace_id ON public.billing_invoice_lines USING btree (namespace, id);


--
-- Name: billinginvoiceline_namespace_invoice_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoiceline_namespace_invoice_id ON public.billing_invoice_lines USING btree (namespace, invoice_id);


--
-- Name: billinginvoiceline_namespace_parent_line_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoiceline_namespace_parent_line_id ON public.billing_invoice_lines USING btree (namespace, parent_line_id);


--
-- Name: billinginvoiceline_namespace_parent_line_id_child_unique_refere; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoiceline_namespace_parent_line_id_child_unique_refere ON public.billing_invoice_lines USING btree (namespace, parent_line_id, child_unique_reference_id) WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));


--
-- Name: billinginvoiceline_namespace_subscription_id_subscription_phase; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoiceline_namespace_subscription_id_subscription_phase ON public.billing_invoice_lines USING btree (namespace, subscription_id, subscription_phase_id, subscription_item_id);


--
-- Name: billinginvoicelinediscount_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicelinediscount_id ON public.billing_invoice_line_discounts USING btree (id);


--
-- Name: billinginvoicelinediscount_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoicelinediscount_namespace ON public.billing_invoice_line_discounts USING btree (namespace);


--
-- Name: billinginvoicelinediscount_namespace_line_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoicelinediscount_namespace_line_id ON public.billing_invoice_line_discounts USING btree (namespace, line_id);


--
-- Name: billinginvoicelinediscount_namespace_line_id_child_unique_refer; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicelinediscount_namespace_line_id_child_unique_refer ON public.billing_invoice_line_discounts USING btree (namespace, line_id, child_unique_reference_id) WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));


--
-- Name: billinginvoicelineusagediscount_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicelineusagediscount_id ON public.billing_invoice_line_usage_discounts USING btree (id);


--
-- Name: billinginvoicelineusagediscount_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoicelineusagediscount_namespace ON public.billing_invoice_line_usage_discounts USING btree (namespace);


--
-- Name: billinginvoicelineusagediscount_namespace_line_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoicelineusagediscount_namespace_line_id ON public.billing_invoice_line_usage_discounts USING btree (namespace, line_id);


--
-- Name: billinginvoicelineusagediscount_namespace_line_id_child_unique_; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicelineusagediscount_namespace_line_id_child_unique_ ON public.billing_invoice_line_usage_discounts USING btree (namespace, line_id, child_unique_reference_id) WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));


--
-- Name: billinginvoicesplitlinegroup_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicesplitlinegroup_id ON public.billing_invoice_split_line_groups USING btree (id);


--
-- Name: billinginvoicesplitlinegroup_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoicesplitlinegroup_namespace ON public.billing_invoice_split_line_groups USING btree (namespace);


--
-- Name: billinginvoicesplitlinegroup_namespace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicesplitlinegroup_namespace_id ON public.billing_invoice_split_line_groups USING btree (namespace, id);


--
-- Name: billinginvoicesplitlinegroup_namespace_unique_reference_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicesplitlinegroup_namespace_unique_reference_id ON public.billing_invoice_split_line_groups USING btree (namespace, unique_reference_id) WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));


--
-- Name: billinginvoiceusagebasedlineconfig_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoiceusagebasedlineconfig_id ON public.billing_invoice_usage_based_line_configs USING btree (id);


--
-- Name: billinginvoiceusagebasedlineconfig_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoiceusagebasedlineconfig_namespace ON public.billing_invoice_usage_based_line_configs USING btree (namespace);


--
-- Name: billinginvoicevalidationissue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicevalidationissue_id ON public.billing_invoice_validation_issues USING btree (id);


--
-- Name: billinginvoicevalidationissue_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billinginvoicevalidationissue_namespace ON public.billing_invoice_validation_issues USING btree (namespace);


--
-- Name: billinginvoicevalidationissue_namespace_invoice_id_dedupe_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billinginvoicevalidationissue_namespace_invoice_id_dedupe_hash ON public.billing_invoice_validation_issues USING btree (namespace, invoice_id, dedupe_hash);


--
-- Name: billingprofile_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingprofile_id ON public.billing_profiles USING btree (id);


--
-- Name: billingprofile_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billingprofile_namespace ON public.billing_profiles USING btree (namespace);


--
-- Name: billingprofile_namespace_default; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingprofile_namespace_default ON public.billing_profiles USING btree (namespace, "default") WHERE ("default" AND (deleted_at IS NULL));


--
-- Name: billingprofile_namespace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingprofile_namespace_id ON public.billing_profiles USING btree (namespace, id);


--
-- Name: billingsequencenumbers_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billingsequencenumbers_namespace ON public.billing_sequence_numbers USING btree (namespace);


--
-- Name: billingsequencenumbers_namespace_scope; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingsequencenumbers_namespace_scope ON public.billing_sequence_numbers USING btree (namespace, scope);


--
-- Name: billingworkflowconfig_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX billingworkflowconfig_id ON public.billing_workflow_configs USING btree (id);


--
-- Name: billingworkflowconfig_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billingworkflowconfig_namespace ON public.billing_workflow_configs USING btree (namespace);


--
-- Name: billingworkflowconfig_namespace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX billingworkflowconfig_namespace_id ON public.billing_workflow_configs USING btree (namespace, id);


--
-- Name: billing_customer_overrides billing_customer_overrides_billing_profiles_billing_customer_ov; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_customer_overrides
    ADD CONSTRAINT billing_customer_overrides_billing_profiles_billing_customer_ov FOREIGN KEY (billing_profile_id) REFERENCES public.billing_profiles(id) ON DELETE SET NULL;


--
-- Name: billing_customer_overrides billing_customer_overrides_customers_billing_customer_override; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_customer_overrides
    ADD CONSTRAINT billing_customer_overrides_customers_billing_customer_override FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


--
-- Name: billing_invoice_line_discounts billing_invoice_line_discounts_billing_invoice_lines_line_amoun; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_line_discounts
    ADD CONSTRAINT billing_invoice_line_discounts_billing_invoice_lines_line_amoun FOREIGN KEY (line_id) REFERENCES public.billing_invoice_lines(id) ON DELETE CASCADE;


--
-- Name: billing_invoice_line_usage_discounts billing_invoice_line_usage_discounts_billing_invoice_lines_line; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_line_usage_discounts
    ADD CONSTRAINT billing_invoice_line_usage_discounts_billing_invoice_lines_line FOREIGN KEY (line_id) REFERENCES public.billing_invoice_lines(id) ON DELETE CASCADE;


--
-- Name: billing_invoice_lines billing_invoice_lines_billing_invoice_flat_fee_line_configs_fla; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_billing_invoice_flat_fee_line_configs_fla FOREIGN KEY (fee_line_config_id) REFERENCES public.billing_invoice_flat_fee_line_configs(id) ON DELETE CASCADE;


--
-- Name: billing_invoice_lines billing_invoice_lines_billing_invoice_lines_detailed_lines; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_billing_invoice_lines_detailed_lines FOREIGN KEY (parent_line_id) REFERENCES public.billing_invoice_lines(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_lines billing_invoice_lines_billing_invoice_split_line_groups_billing; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_billing_invoice_split_line_groups_billing FOREIGN KEY (split_line_group_id) REFERENCES public.billing_invoice_split_line_groups(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_lines billing_invoice_lines_billing_invoice_usage_based_line_configs_; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_billing_invoice_usage_based_line_configs_ FOREIGN KEY (usage_based_line_config_id) REFERENCES public.billing_invoice_usage_based_line_configs(id) ON DELETE CASCADE;


--
-- Name: billing_invoice_lines billing_invoice_lines_billing_invoices_billing_invoice_lines; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_billing_invoices_billing_invoice_lines FOREIGN KEY (invoice_id) REFERENCES public.billing_invoices(id) ON DELETE CASCADE;


--
-- Name: billing_invoice_lines billing_invoice_lines_subscription_items_billing_lines; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_subscription_items_billing_lines FOREIGN KEY (subscription_item_id) REFERENCES public.subscription_items(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_lines billing_invoice_lines_subscription_phases_billing_lines; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_subscription_phases_billing_lines FOREIGN KEY (subscription_phase_id) REFERENCES public.subscription_phases(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_lines billing_invoice_lines_subscriptions_billing_lines; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_lines
    ADD CONSTRAINT billing_invoice_lines_subscriptions_billing_lines FOREIGN KEY (subscription_id) REFERENCES public.subscriptions(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_split_line_groups billing_invoice_split_line_groups_subscription_items_billing_sp; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_split_line_groups
    ADD CONSTRAINT billing_invoice_split_line_groups_subscription_items_billing_sp FOREIGN KEY (subscription_item_id) REFERENCES public.subscription_items(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_split_line_groups billing_invoice_split_line_groups_subscription_phases_billing_s; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_split_line_groups
    ADD CONSTRAINT billing_invoice_split_line_groups_subscription_phases_billing_s FOREIGN KEY (subscription_phase_id) REFERENCES public.subscription_phases(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_split_line_groups billing_invoice_split_line_groups_subscriptions_billing_split_l; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_split_line_groups
    ADD CONSTRAINT billing_invoice_split_line_groups_subscriptions_billing_split_l FOREIGN KEY (subscription_id) REFERENCES public.subscriptions(id) ON DELETE SET NULL;


--
-- Name: billing_invoice_validation_issues billing_invoice_validation_issues_billing_invoices_billing_invo; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoice_validation_issues
    ADD CONSTRAINT billing_invoice_validation_issues_billing_invoices_billing_invo FOREIGN KEY (invoice_id) REFERENCES public.billing_invoices(id) ON DELETE CASCADE;


--
-- Name: billing_invoices billing_invoices_apps_billing_invoice_invoicing_app; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_apps_billing_invoice_invoicing_app FOREIGN KEY (invoicing_app_id) REFERENCES public.apps(id);


--
-- Name: billing_invoices billing_invoices_apps_billing_invoice_payment_app; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_apps_billing_invoice_payment_app FOREIGN KEY (payment_app_id) REFERENCES public.apps(id);


--
-- Name: billing_invoices billing_invoices_apps_billing_invoice_tax_app; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_apps_billing_invoice_tax_app FOREIGN KEY (tax_app_id) REFERENCES public.apps(id);


--
-- Name: billing_invoices billing_invoices_billing_profiles_billing_invoices; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_billing_profiles_billing_invoices FOREIGN KEY (source_billing_profile_id) REFERENCES public.billing_profiles(id);


--
-- Name: billing_invoices billing_invoices_billing_workflow_configs_billing_invoices; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_billing_workflow_configs_billing_invoices FOREIGN KEY (workflow_config_id) REFERENCES public.billing_workflow_configs(id);


--
-- Name: billing_invoices billing_invoices_customers_billing_invoice; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_invoices
    ADD CONSTRAINT billing_invoices_customers_billing_invoice FOREIGN KEY (customer_id) REFERENCES public.customers(id);


--
-- Name: billing_profiles billing_profiles_apps_billing_profile_invoicing_app; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_profiles
    ADD CONSTRAINT billing_profiles_apps_billing_profile_invoicing_app FOREIGN KEY (invoicing_app_id) REFERENCES public.apps(id);


--
-- Name: billing_profiles billing_profiles_apps_billing_profile_payment_app; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_profiles
    ADD CONSTRAINT billing_profiles_apps_billing_profile_payment_app FOREIGN KEY (payment_app_id) REFERENCES public.apps(id);


--
-- Name: billing_profiles billing_profiles_apps_billing_profile_tax_app; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_profiles
    ADD CONSTRAINT billing_profiles_apps_billing_profile_tax_app FOREIGN KEY (tax_app_id) REFERENCES public.apps(id);


--
-- Name: billing_profiles billing_profiles_billing_workflow_configs_billing_profile; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_profiles
    ADD CONSTRAINT billing_profiles_billing_workflow_configs_billing_profile FOREIGN KEY (workflow_config_id) REFERENCES public.billing_workflow_configs(id);


--
-- PostgreSQL database dump complete
--

