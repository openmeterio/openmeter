package entutils

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

func MapPaged[I, O any](paged pagination.PagedResponse[I], mapper func(I) O) pagination.PagedResponse[O] {
	return pagination.PagedResponse[O]{
		TotalCount: paged.TotalCount,
		Items: lo.Map(paged.Items, func(item I, _ int) O {
			return mapper(item)
		}),
		Page: paged.Page,
	}
}
