
--- Source: https://github.com/geckoboard/pgulid/blob/master/pgulid.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Note: pg_temp only exists for the current session, so we don't need to clean up the function after the migration is done
CREATE OR REPLACE FUNCTION pg_temp.generate_ulid()
RETURNS TEXT
AS $$
DECLARE
  -- Crockford's Base32
  encoding   BYTEA = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
  timestamp  BYTEA = E'\\000\\000\\000\\000\\000\\000';
  output     TEXT = '';

  unix_time  BIGINT;
  ulid       BYTEA;
BEGIN
  -- 6 timestamp bytes
  unix_time = (EXTRACT(EPOCH FROM CLOCK_TIMESTAMP()) * 1000)::BIGINT;
  timestamp = SET_BYTE(timestamp, 0, (unix_time >> 40)::BIT(8)::INTEGER);
  timestamp = SET_BYTE(timestamp, 1, (unix_time >> 32)::BIT(8)::INTEGER);
  timestamp = SET_BYTE(timestamp, 2, (unix_time >> 24)::BIT(8)::INTEGER);
  timestamp = SET_BYTE(timestamp, 3, (unix_time >> 16)::BIT(8)::INTEGER);
  timestamp = SET_BYTE(timestamp, 4, (unix_time >> 8)::BIT(8)::INTEGER);
  timestamp = SET_BYTE(timestamp, 5, unix_time::BIT(8)::INTEGER);

  -- 10 entropy bytes
  ulid = timestamp || gen_random_bytes(10);

  -- Encode the timestamp
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 0) & 224) >> 5));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 0) & 31)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 1) & 248) >> 3));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 1) & 7) << 2) | ((GET_BYTE(ulid, 2) & 192) >> 6)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 2) & 62) >> 1));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 2) & 1) << 4) | ((GET_BYTE(ulid, 3) & 240) >> 4)));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 3) & 15) << 1) | ((GET_BYTE(ulid, 4) & 128) >> 7)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 4) & 124) >> 2));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 4) & 3) << 3) | ((GET_BYTE(ulid, 5) & 224) >> 5)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 5) & 31)));

  -- Encode the entropy
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 6) & 248) >> 3));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 6) & 7) << 2) | ((GET_BYTE(ulid, 7) & 192) >> 6)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 7) & 62) >> 1));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 7) & 1) << 4) | ((GET_BYTE(ulid, 8) & 240) >> 4)));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 8) & 15) << 1) | ((GET_BYTE(ulid, 9) & 128) >> 7)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 9) & 124) >> 2));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 9) & 3) << 3) | ((GET_BYTE(ulid, 10) & 224) >> 5)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 10) & 31)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 11) & 248) >> 3));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 11) & 7) << 2) | ((GET_BYTE(ulid, 12) & 192) >> 6)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 12) & 62) >> 1));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 12) & 1) << 4) | ((GET_BYTE(ulid, 13) & 240) >> 4)));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 13) & 15) << 1) | ((GET_BYTE(ulid, 14) & 128) >> 7)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 14) & 124) >> 2));
  output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 14) & 3) << 3) | ((GET_BYTE(ulid, 15) & 224) >> 5)));
  output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 15) & 31)));

  RETURN output;
END
$$
LANGUAGE plpgsql
VOLATILE;

-- Recommended to create a backup table, but we don't have a way to revert this migration.
--
-- atlas:nolint PG110
CREATE TABLE IF NOT EXISTS billing_backup_migrated_flat_fees
    AS
    SELECT l.*, c.per_unit_amount, c.category, c.payment_term, c.index
    FROM billing_invoice_lines l JOIN billing_invoice_flat_fee_line_configs c ON (l.fee_line_config_id = c.id)
    WHERE type = 'flat_fee' AND status = 'valid';

CREATE OR REPLACE FUNCTION pg_temp.migrate_flat_fees_to_ubp_flat_fees(idToMigrate TEXT)
RETURNS TEXT
AS $$
DECLARE
    existing_flat_fee RECORD;
    new_usage_based_line billing_invoice_lines%ROWTYPE;
    ubp_flat_fee_config_id TEXT;
    ubp_line_id TEXT;
    updated_metadata jsonb;
    metadata_in jsonb;
BEGIN
    SELECT
        l.*,
        c.per_unit_amount, c.payment_term
    INTO existing_flat_fee
    FROM
        billing_invoice_lines l JOIN billing_invoice_flat_fee_line_configs c ON (l.fee_line_config_id = c.id)
    WHERE
        l.id = idToMigrate;

    IF existing_flat_fee IS NULL THEN
        RAISE EXCEPTION 'Flat fee line with id % not found', idToMigrate;
    END IF;

    -- let's generate a new usagebased line config first

    ubp_flat_fee_config_id = pg_temp.generate_ulid();

    INSERT INTO
        billing_invoice_usage_based_line_configs (id, namespace, price_type, price, pre_line_period_quantity, metered_quantity, metered_pre_line_period_quantity)
    VALUES (
        ubp_flat_fee_config_id,
        existing_flat_fee.namespace,
        'flat',
        format('{"type": "flat", "amount": "%s", "paymentTerm": "%s"}', existing_flat_fee.per_unit_amount, existing_flat_fee.payment_term)::jsonb,
        0, -- pre_line_period_quantity
        1, -- metered_quantity
        0 -- metered_pre_line_period_quantity
    );

    -- let's create a new usagebased line

    ubp_line_id = pg_temp.generate_ulid();
    IF existing_flat_fee.metadata = 'null'::jsonb THEN
        metadata_in = '{}'::jsonb;
    ELSE
        metadata_in = coalesce(existing_flat_fee.metadata, '{}'::jsonb);
    END IF;

    updated_metadata = jsonb_insert(metadata_in, '{/openmeter-line-reason}', '"add-line-wrapping"');
    INSERT INTO billing_invoice_lines (
        id,
        namespace,
        metadata,
        created_at,
        updated_at,
        deleted_at,
        name,
        description,
        period_start,
        period_end,
        invoice_at,
        type,
        status,
        currency,
        quantity,
        tax_config,
        invoice_id,
        fee_line_config_id,
        usage_based_line_config_id,
        parent_line_id,
        child_unique_reference_id,
        amount,
        taxes_total,
        taxes_inclusive_total,
        taxes_exclusive_total,
        charges_total,
        discounts_total,
        total,
        invoicing_app_external_id,
        subscription_id,
        subscription_item_id,
        subscription_phase_id,
        line_ids,
        managed_by,
        ratecard_discounts)
    VALUES (
        ubp_line_id,
        existing_flat_fee.namespace,
        updated_metadata,
        existing_flat_fee.created_at,
        existing_flat_fee.updated_at,
        existing_flat_fee.deleted_at,
        existing_flat_fee.name,
        existing_flat_fee.description,
        existing_flat_fee.period_start,
        existing_flat_fee.period_end,
        existing_flat_fee.invoice_at,
        'usage_based',
        'valid',
        existing_flat_fee.currency,
        existing_flat_fee.quantity,
        existing_flat_fee.tax_config,
        existing_flat_fee.invoice_id,
        null, -- fee_line_config_id
        ubp_flat_fee_config_id,
        existing_flat_fee.parent_line_id, -- parent_line_id
        existing_flat_fee.child_unique_reference_id,
        existing_flat_fee.amount,
        existing_flat_fee.taxes_total,
        existing_flat_fee.taxes_inclusive_total,
        existing_flat_fee.taxes_exclusive_total,
        existing_flat_fee.charges_total,
        existing_flat_fee.discounts_total,
        existing_flat_fee.total,
        NULL, -- invoicing_app_external_id (the flat_fee detailed line is syncronized to external systems not this specific line)
        existing_flat_fee.subscription_id,
        existing_flat_fee.subscription_item_id,
        existing_flat_fee.subscription_phase_id,
        existing_flat_fee.line_ids,
        existing_flat_fee.managed_by,
        existing_flat_fee.ratecard_discounts
    );

    -- let's convert the flat fee line into a detailed line

    UPDATE billing_invoice_lines SET
        status = 'detailed',
        parent_line_id = ubp_line_id,
        child_unique_reference_id = 'flat-price', -- FlatPriceChildUniqueReferenceID
        subscription_id = NULL,
        subscription_item_id = NULL,
        subscription_phase_id = NULL,
        metadata = NULL,
        managed_by = 'system'
    WHERE
        id = idToMigrate;

    RETURN ubp_line_id;
END
$$
LANGUAGE plpgsql
VOLATILE;


--- Let's do the migraiton

SELECT pg_temp.migrate_flat_fees_to_ubp_flat_fees(id) FROM billing_invoice_lines WHERE type = 'flat_fee' AND status = 'valid';
