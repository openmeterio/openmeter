-- reverse: create index "rteinvalidation_source_event_id" to table: "rte_invalidations"
DROP INDEX "rteinvalidation_source_event_id";
-- reverse: create index "rte_invalidation_retention" to table: "rte_invalidations"
DROP INDEX "rte_invalidation_retention";
-- reverse: create "rte_invalidations" table
DROP TABLE "rte_invalidations";
-- reverse: create index "rtedemand_namespace" to table: "rte_demands"
DROP INDEX "rtedemand_namespace";
-- reverse: create index "rte_demand_work" to table: "rte_demands"
DROP INDEX "rte_demand_work";
-- reverse: create index "rte_demand_owner" to table: "rte_demands"
DROP INDEX "rte_demand_owner";
-- reverse: create index "rte_demand_identity" to table: "rte_demands"
DROP INDEX "rte_demand_identity";
-- reverse: create "rte_demands" table
DROP TABLE "rte_demands";
