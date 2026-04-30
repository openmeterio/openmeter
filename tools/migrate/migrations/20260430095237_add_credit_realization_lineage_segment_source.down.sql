-- reverse: modify "credit_realization_lineage_segments" table
ALTER TABLE "credit_realization_lineage_segments" DROP COLUMN "source_backing_transaction_group_id", DROP COLUMN "source_state";
