# Credit filters

Credit restrictions use a versioned JSON envelope. An empty feature list imposes
no restriction. Unknown versions and fields fail decoding so a newer restriction
cannot silently disappear in an older reader.

During the storage transition, writers populate both the JSON envelope and the
legacy feature columns. Readers still use the legacy columns. Deploy these
writers everywhere before backfilling and switching readers to JSON.

`StoredFilters` retains an explicit `FiltersVersion`; version-specific readers
and writers dispatch through it. V1 is a frozen feature-only format. `Filters`
contains the common matching dimensions, and its JSON writer selects the oldest
format capable of representing them. Unknown versions and dimensions are rejected.
