-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "feature_filters" text[] NULL;

-- modify "credit_realization_lineages" table
ALTER TABLE "credit_realization_lineages" ADD COLUMN "advance_features" text[] NULL;
