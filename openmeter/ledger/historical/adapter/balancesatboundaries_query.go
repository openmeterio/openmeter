package adapter

import (
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type balancesAtBoundariesQuery struct {
	input ledger.GetBalancesAtBoundariesInput
}

func (q balancesAtBoundariesQuery) SQL() (string, []any, error) {
	var combined *sql.Selector
	for idx, query := range q.input.Queries {
		selector, err := (&sumEntriesQuery{query: query}).selector()
		if err != nil {
			return "", nil, err
		}
		selector.AppendSelectExpr(sql.ExprFunc(func(b *sql.Builder) {
			b.Arg(idx).WriteString("::integer AS boundary_index")
		}))
		if combined == nil {
			combined = selector
		} else {
			combined.UnionAll(selector)
		}
	}
	query, args := combined.Query()
	return query, args, nil
}
