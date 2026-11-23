-- create "rte_demand_locks" table
CREATE TABLE "rte_demand_locks" (
  "id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "rte_demand_locks_rte_demands_demand" FOREIGN KEY ("id") REFERENCES "rte_demands" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Demand can be registered by workers running either side of this migration.
-- Create its lock in the same transaction so every committed demand is claimable.
CREATE FUNCTION rte_create_demand_lock() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO rte_demand_locks (id) VALUES (NEW.id);
  RETURN NEW;
END;
$$;
CREATE TRIGGER rte_create_demand_lock
  AFTER INSERT ON rte_demands
  FOR EACH ROW EXECUTE FUNCTION rte_create_demand_lock();

INSERT INTO rte_demand_locks (id)
SELECT id FROM rte_demands;
