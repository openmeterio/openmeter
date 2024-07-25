package entutils

import (
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func GetOrdering(order sortx.Order) []sql.OrderTermOption {
	type o = sql.OrderTermOption

	switch order {
	case sortx.OrderAsc:
		return []o{sql.OrderAsc()}
	case sortx.OrderDesc:
		return []o{sql.OrderDesc()}
	default:
		return getStrOrdering(string(order))
	}
}

func getStrOrdering(order string) []sql.OrderTermOption {
	type o = sql.OrderTermOption

	switch order {
	case string(sortx.OrderAsc):
		return []o{sql.OrderAsc()}
	case string(sortx.OrderDesc):
		return []o{sql.OrderDesc()}
	default:
		return []o{}
	}
}
