-- reverse: create index "creditrealizationlineagesegment_lineage_id_closed_at" to table: "credit_realization_lineage_segments"
DROP INDEX "creditrealizationlineagesegment_lineage_id_closed_at";
-- reverse: create index "creditrealizationlineagesegment_lineage_id" to table: "credit_realization_lineage_segments"
DROP INDEX "creditrealizationlineagesegment_lineage_id";
-- reverse: create index "creditrealizationlineagesegment_id" to table: "credit_realization_lineage_segments"
DROP INDEX "creditrealizationlineagesegment_id";
-- reverse: create "credit_realization_lineage_segments" table
DROP TABLE "credit_realization_lineage_segments";
-- reverse: create index "creditrealizationlineage_namespace_root_realization_id" to table: "credit_realization_lineages"
DROP INDEX "creditrealizationlineage_namespace_root_realization_id";
-- reverse: create index "creditrealizationlineage_namespace_customer_id" to table: "credit_realization_lineages"
DROP INDEX "creditrealizationlineage_namespace_customer_id";
-- reverse: create index "creditrealizationlineage_namespace" to table: "credit_realization_lineages"
DROP INDEX "creditrealizationlineage_namespace";
-- reverse: create index "creditrealizationlineage_id" to table: "credit_realization_lineages"
DROP INDEX "creditrealizationlineage_id";
-- reverse: create "credit_realization_lineages" table
DROP TABLE "credit_realization_lineages";
