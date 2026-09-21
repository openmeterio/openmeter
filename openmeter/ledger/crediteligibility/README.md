# Credit filters

Credit restrictions use a versioned JSON envelope. An empty feature list imposes
no restriction. Unknown versions and fields fail decoding so a newer restriction
cannot silently disappear in an older reader.

Readers use JSON exclusively. Writers still populate the legacy feature columns
so instances of the previous release can coexist during deployment. Stop these
dual writes only after every API instance and worker uses the JSON readers.

The cutover migration requires all API instances and workers to run the prior
dual-writing release. It fills missing envelopes before making them required.
Route IDs, routing keys, and ledger entries are unchanged. Before rolling back
to feature-column readers, run the cutover down migration to rebuild their
projections; it rejects unsupported versions and envelopes containing other
dimensions.

`Filters.Version` is retained by Ent, normalization, and JSON round trips.
Writers explicitly choose v1 when creating filters; encoding honors that version.
The codec switches on the version and currently supports only v1.
Missing and unsupported versions are rejected; there is no implicit upgrade.
