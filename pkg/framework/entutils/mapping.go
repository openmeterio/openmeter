package entutils

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

func MapPagedWithErr[I, O any](paged pagination.PagedResponse[I], mapper func(I) (O, error)) (pagination.PagedResponse[O], error) {
	items, err := slicesx.MapWithErr(paged.Items, mapper)
	if err != nil {
		return pagination.PagedResponse[O]{}, err
	}

	return pagination.PagedResponse[O]{
		TotalCount: paged.TotalCount,
		Items:      items,
		Page:       paged.Page,
	}, nil
}
