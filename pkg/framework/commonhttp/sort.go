package commonhttp

import (
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func GetSortOrder[TInput comparable](asc TInput, inp *TInput) sortx.Order {
	return defaultx.WithDefault(
		convert.SafeDeRef[TInput, sortx.Order](
			inp,
			func(o TInput) *sortx.Order {
				if o == asc {
					return convert.ToPointer(sortx.OrderAsc)
				}
				return convert.ToPointer(sortx.OrderDesc)
			},
		),
		sortx.OrderNone,
	)
}
