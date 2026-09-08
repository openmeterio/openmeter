// Package legacylineage maintains mutable credit-realization roots and amount
// segments for pre-cutover collections. Existing histories still require its
// reads and writes for backfill, recognition, and correction.
//
// Deprecated: New collections use immutable ledger origins and derive remaining
// amounts from ledger entries. Do not create lineage state for origin-tracked
// collections. Retain this package until legacy histories no longer need it.
package legacylineage
