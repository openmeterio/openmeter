-- drop index "balancesnapshot_namespace_at" from table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_at";
-- drop index "balancesnapshot_namespace_balance" from table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_balance";
-- drop index "balancesnapshot_namespace_balance_at" from table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_balance_at";
-- create index "balancesnapshot_namespace_owner_id_at" to table: "balance_snapshots"
CREATE INDEX "balancesnapshot_namespace_owner_id_at" ON "balance_snapshots" ("namespace", "owner_id", "at") WHERE (deleted_at IS NULL);
