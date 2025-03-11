-- reverse: create index "balancesnapshot_namespace_owner_id_at" to table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_owner_id_at";
-- reverse: drop index "balancesnapshot_namespace_balance_at" from table: "balance_snapshots"
CREATE INDEX "balancesnapshot_namespace_balance_at" ON "balance_snapshots" ("namespace", "balance", "at");
-- reverse: drop index "balancesnapshot_namespace_balance" from table: "balance_snapshots"
CREATE INDEX "balancesnapshot_namespace_balance" ON "balance_snapshots" ("namespace", "balance");
-- reverse: drop index "balancesnapshot_namespace_at" from table: "balance_snapshots"
CREATE INDEX "balancesnapshot_namespace_at" ON "balance_snapshots" ("namespace", "at");
