-- modify "credit_realization_lineage_segments" table
ALTER TABLE "credit_realization_lineage_segments" ADD COLUMN "source_state" character varying NULL, ADD COLUMN "source_backing_transaction_group_id" character(26) NULL;
