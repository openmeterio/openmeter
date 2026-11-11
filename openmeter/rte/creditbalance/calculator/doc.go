// Package calculator adapts the current customer credit balance calculation to RTE.
//
// Calculate leaves the source's AsOf unset so live, unbooked charge usage is
// included. RTE's calculation start orders stored results. Passing it to the
// source as a historical cutoff would change the meaning of Live.
//
// Each calculator instance owns a bounded per-charge rating cache. The source
// still reads current quantities and charge inputs on every execution. Identical
// rating inputs can reuse a successful total across calls on that instance.
package calculator
