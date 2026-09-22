# Credit filters

Credit restrictions use a versioned JSON envelope. An empty feature list imposes
no restriction. Unknown versions and fields fail decoding so a newer restriction
cannot silently disappear in an older reader.

Readers use JSON exclusively. Writers maintain the legacy feature columns for
compatibility with older readers.

`Filters.Version` is retained by Ent, normalization, and JSON round trips.
Writers explicitly choose v1 when creating filters; encoding honors that version.
The codec switches on the version and currently supports only v1.
Missing and unsupported versions are rejected; there is no implicit upgrade.
