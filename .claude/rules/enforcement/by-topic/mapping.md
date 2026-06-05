# Enforcement: mapping (2 rules)

Topic file. Loaded on demand when an agent works on something in the `mapping` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `name-convert-001` — Type-translation functions use FromAPI.../ToAPI.../FromDB.../ToDB... names and map/mapped terminology

*source: `deep_scan`*

**Why:** Domain↔API↔DB conversions use goverter (convert.go → convert.gen.go) and goderive (derived.gen.go). Hand-written conversion files/functions follow the FromAPI.../ToAPI.../FromDB.../ToDB... naming convention (the /go-types-conversion skill) and use map/mapped terminology, never project/projected. Generated *.gen.go files carry DO-NOT-EDIT headers.

### `sem-entitymapping-001` — Use generated goverter/goderive converters for domain↔API↔DB mapping rather than hand-writing boilerplate

*source: `deep_scan`*

**Why:** convert.gen.go files are generated from goverter converter interfaces declared in convert.go; billing/derived.gen.go is generated from goderive annotations. Translating between domain, API, and DB representations should use these generated converters, with hand-written helpers only following the FromAPI/ToAPI/FromDB/ToDB naming convention. Never edit *.gen.go (DO-NOT-EDIT header).
