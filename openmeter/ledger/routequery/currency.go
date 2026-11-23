package routequery

import "entgo.io/ent/dialect/sql"

// CustomCurrencyID selects the ID from a ledger route's currency column.
// CurrencyReference.MarshalText stores a custom reference as custom|v1|code|id|precision.
// A fiat code has no fourth part, so PostgreSQL returns an empty string.
func CustomCurrencyID(column string) sql.Querier {
	return sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("split_part").Wrap(func(b *sql.Builder) {
			b.Ident(column).Comma().Arg("|").Comma().Arg(4)
		})
	})
}
