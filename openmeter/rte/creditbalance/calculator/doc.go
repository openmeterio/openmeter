// Package calculator adapts the current customer credit balance calculation to RTE.
//
// Calculate leaves the source's AsOf unset so live, unbooked charge usage is
// included. RTE's calculation start orders stored results. Passing it to the
// source as a historical cutoff would change the meaning of Live.
package calculator
