CREATE EXTENSION IF NOT EXISTS pgcrypto;

--
-- Note: we'll create permanent functions (that we do not later drop). The reasoning is twofold:
-- 1. Somewhat self-evidently for later reuse
-- 2. So we have access to them test-time
-- All functions will be prefixed with "om_func_"
--

CREATE OR REPLACE FUNCTION om_func_generate_ulid()
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
LANGUAGE plpgsql VOLATILE;


CREATE OR REPLACE FUNCTION om_func_go_add_date_normalized(
    base TIMESTAMPTZ,
    duration INTERVAL
) RETURNS TIMESTAMPTZ AS $$
DECLARE
    result_date            DATE;
    dur_days               INT;
    dur_months             INT;
    dur_years              INT;
    dur_time_part          INTERVAL;
    days_in_target_month   INT;
    target_year            INT;
    target_month           INT;
    target_day             INT;
    target_time_part       INTERVAL;
    overflow_days          INT;
BEGIN
    -- We're proceeding in good faith that the input duration is normalized according to our needs, i.e.
    -- - No fractional components
    -- - Normalized so no components overflow (e.g. 13 months)

    -- Store the duration components
    dur_days = EXTRACT(DAY FROM duration);
    dur_months = EXTRACT(MONTH FROM duration);
    dur_years = EXTRACT(YEAR FROM duration);
    dur_time_part = duration - date_trunc('day', duration);

    -- Calculate target year and month
    target_year = EXTRACT(YEAR FROM base) + dur_years;
    target_month = EXTRACT(MONTH FROM base) + dur_months;
    target_day = EXTRACT(DAY FROM base) + dur_days;
    target_time_part = base - date_trunc('day', base);

    -- Handle month overflow/underflow
    WHILE target_month > 12 LOOP
        target_month = target_month - 12;
        target_year = target_year + 1;
    END LOOP;

    WHILE target_month < 1 LOOP
        target_month = target_month + 12;
        target_year = target_year - 1;
    END LOOP;

    -- Get the number of days in the target month
    days_in_target_month = EXTRACT(
        DAY FROM
        (
            DATE_TRUNC('MONTH', MAKE_DATE(target_year, target_month, 1))
            + INTERVAL '1 MONTH - 1 DAY'
        )::DATE
    );

    -- Check if the target day exists in the target month
    -- If it does, use it
    IF target_day <= days_in_target_month THEN
        result_date = MAKE_DATE(target_year, target_month, target_day);
    ELSE
    -- If it does not, normalize it
        overflow_days = target_day - days_in_target_month;
        result_date = MAKE_DATE(target_year, target_month, days_in_target_month) + overflow_days;
    END IF;

    -- Finally, lets add back the time part (there's no normalization there so we can just add the durations)
    RETURN (result_date + dur_time_part + target_time_part)::TIMESTAMPTZ;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Normalizes the interval to days + time components
CREATE OR REPLACE FUNCTION om_func_go_normalize_interval_to_str(
    start_ts TIMESTAMPTZ,
    end_ts TIMESTAMPTZ
) RETURNS TEXT AS $$
BEGIN
    SET intervalstyle = 'iso_8601'; -- Function scoped change

    RETURN (end_ts - start_ts)::TEXT;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- This replicates the old behavior of recurrence.Recurrence
CREATE OR REPLACE FUNCTION om_func_get_go_normalized_last_iteration_not_after_cutoff(
    anchor TIMESTAMPTZ,
    duration INTERVAL,
    cutoff TIMESTAMPTZ
) RETURNS TIMESTAMPTZ AS $$
DECLARE
    result_ts TIMESTAMPTZ;
    iter_ts TIMESTAMPTZ;
BEGIN
    result_ts = anchor;
    iter_ts = anchor;

    IF result_ts <= cutoff THEN
        -- We iterate onwards
        WHILE iter_ts <= cutoff LOOP
            result_ts = iter_ts;

            iter_ts = om_func_go_add_date_normalized(iter_ts, duration);
        END LOOP;
    ELSE
        -- We iterate backwards
        WHILE iter_ts > cutoff LOOP
            iter_ts = om_func_go_add_date_normalized(iter_ts, duration * -1);

            result_ts = iter_ts;
        END LOOP;
    END IF;

    return result_ts;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- This is the main method executing the calculations
CREATE OR REPLACE FUNCTION om_func_update_usage_period_durations(
    ent_id TEXT,
    cutoff TIMESTAMPTZ
) RETURNS VOID AS $$
DECLARE
    current_entitlement RECORD;
    current_usage_reset RECORD;
    initial_usage_reset RECORD;
    iteration_ts        TIMESTAMP;
    next_iteration_ts   TIMESTAMP;
    normalized_interval INTERVAL;
    current_anchor TIMESTAMPTZ;
    entitlement_usage_reset_iteration INT;
    usage_reset_iteration INT;
BEGIN
    SELECT * FROM entitlements WHERE id = ent_id INTO current_entitlement;

    entitlement_usage_reset_iteration = 0;

    -- We need to create an initial "virtual" usage reset
    SELECT
        current_entitlement.measure_usage_from AS reset_time,
        current_entitlement.usage_period_anchor AS anchor,
        current_entitlement.usage_period_interval AS usage_period_interval
    INTO initial_usage_reset;

    -- We have to build a timeline of usage resets
    FOR current_usage_reset IN
        SELECT
            reset_time,
            anchor,
            usage_period_interval,
            LEAD(reset_time) OVER w AS next_reset_time
        FROM (
            SELECT
                initial_usage_reset.reset_time AS reset_time,
                initial_usage_reset.anchor AS anchor,
                initial_usage_reset.usage_period_interval AS usage_period_interval
            UNION ALL
            SELECT
                reset_time, anchor, usage_period_interval
            FROM usage_resets
            WHERE entitlement_id = current_entitlement.id
            ORDER BY reset_time ASC
        ) eur
        WINDOW w AS (
            ORDER BY reset_time ASC
        )
    LOOP
        -- The iteration starting point will be the closest iteration of the current anchor not after the reset time
        iteration_ts = om_func_get_go_normalized_last_iteration_not_after_cutoff(
            current_usage_reset.anchor,
            current_usage_reset.usage_period_interval::INTERVAL,
            current_usage_reset.reset_time
        );

        -- And let's zero all the variables we'll use
        next_iteration_ts = NULL::TIMESTAMP;
        usage_reset_iteration = 0;

        WHILE TRUE LOOP
            IF iteration_ts > cutoff THEN
                EXIT;
            END IF;

            IF current_usage_reset.next_reset_time IS NOT NULL AND iteration_ts > current_usage_reset.next_reset_time THEN
                EXIT;
            END IF;

            -- Now we'll calculate the next iteration timestamp via the normalized function
            next_iteration_ts = om_func_go_add_date_normalized(
                iteration_ts::TIMESTAMPTZ,
                current_usage_reset.usage_period_interval::INTERVAL
            );

            -- Then we'll normalize the interval between the two
            normalized_interval = om_func_go_normalize_interval_to_str(iteration_ts, next_iteration_ts);

            -- If this is our first iteration for the usage reset, we need to UPDATE the usage reset
            IF usage_reset_iteration = 0 AND entitlement_usage_reset_iteration > 0 THEN
                -- Only update if this is not the initial virtual usage reset

                UPDATE usage_resets
                SET
                    usage_period_interval = normalized_interval,
                    -- We need to update the anchor time to be the closest occurence of the old anchor time not after the reset time (calculated via the old interval and algo)
                    anchor = om_func_get_go_normalized_last_iteration_not_after_cutoff(
                        usage_resets.anchor,
                        usage_resets.usage_period_interval::INTERVAL,
                        usage_resets.reset_time
                    )
                WHERE entitlement_id = current_entitlement.id
                AND reset_time = current_usage_reset.reset_time;
            ELSE
                current_anchor = iteration_ts;

                -- If we're in the "current" period relative to the cutoff
                -- meaning iteration_ts is before the cutoff and next_iteration_ts is after it,
                -- then the new algo should take place (without normalization)
                -- so we should restore the usage_period_interval to the original value...
                IF next_iteration_ts > cutoff THEN
                    normalized_interval = current_usage_reset.usage_period_interval;
                    current_anchor = current_usage_reset.anchor;
                    -- Note that we don't need to exit here as the loop will stop at the start of next iteration
                END IF;

            -- Otherwise, we need to INSERT a new usage reset record
                INSERT INTO usage_resets (
                    namespace,
                    id,
                    created_at,
                    updated_at,
                    entitlement_id,
                    reset_time,
                    anchor,
                    usage_period_interval
                )
                VALUES (
                    current_entitlement.namespace,
                    om_func_generate_ulid(),
                    NOW(),
                    NOW(),
                    current_entitlement.id,
                    iteration_ts,
                    current_anchor,
                    normalized_interval
                );
            END IF;

            iteration_ts = next_iteration_ts;
            usage_reset_iteration = usage_reset_iteration + 1;
        END LOOP;

        entitlement_usage_reset_iteration = entitlement_usage_reset_iteration + 1;
    END LOOP;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Let's run the migration
SELECT om_func_update_usage_period_durations(id, NOW()) FROM entitlements WHERE entitlement_type = 'metered';
