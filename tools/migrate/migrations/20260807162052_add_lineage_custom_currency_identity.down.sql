-- reverse: modify "credit_realization_lineages" table
--
-- The added "custom_currency_id" column is dropped, but the widened
-- "currency" column is intentionally left at varchar(24) rather than
-- narrowed back to varchar(3). Narrowing is lossy and would fail outright
-- once any row stores a custom currency code longer than 3 characters.
-- Application code from before this migration only ever writes 3-character
-- fiat codes, so it remains fully compatible with the wider column.
ALTER TABLE "credit_realization_lineages" DROP COLUMN "custom_currency_id";
