-- reverse: modify "rte_demands" table
ALTER TABLE "rte_demands" ADD COLUMN "refresh_interval" bigint NOT NULL DEFAULT 0;
