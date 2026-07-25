# Frozen Ent migration baseline

This package is the one-time compatibility bridge from runtime Ent schema migration to versioned migrations.

- OpenMeter commit: `12ab7b082035f2f93972c7f98973c5502107c157`
- Migration baseline: `20260709134422`
- PostgreSQL schema descriptors: frozen from `openmeter/ent/db/migrate/schema.go` at that commit

Do not regenerate or update `schema.go` when the current Ent schema changes. `openmeter-jobs migrate adopt-ent` migrates a non-empty OpenMeter database without `schema_om` using these frozen descriptors, applies the ordered reconciliation scripts, and records the baseline version. It deliberately stops there. Run `openmeter-jobs migrate` afterward to upgrade through the normal migration history to the target OpenMeter version.

The reconciliation scripts contain database state that Ent table descriptors cannot provide, including persistent functions, triggers, views, and singleton rows. They must remain rerunnable because adoption can be interrupted after reconciliation but before `schema_om` records the baseline.

Databases that already contain `schema_om`, including versions older than the baseline, must never use this package. They continue through the normal historical migrations from their recorded version. Because adoption always establishes the same baseline, the procedure is independent of the target OpenMeter version as long as that target contains this frozen bridge and the migrations after the baseline.
