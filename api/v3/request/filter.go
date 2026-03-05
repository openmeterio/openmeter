package request

import (
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func convertContainsOperator(value *string) *string {
	if value == nil {
		return nil
	}

	return lo.ToPtr(filter.ContainsPattern(*value))
}

func convertQueryFilterStringList(source *[]api.QueryFilterString) *[]filter.FilterString {
	if source == nil {
		return nil
	}

	items := make([]filter.FilterString, len(*source))
	for i := range *source {
		items[i] = ConvertQueryFilterString((*source)[i])
	}

	return &items
}

// ConvertQueryFilterString converts an API QueryFilterString to internal filter.FilterString.
func ConvertQueryFilterString(source api.QueryFilterString) filter.FilterString {
	return filter.FilterString{
		Eq:    source.Eq,
		Ne:    source.Neq,
		In:    source.In,
		Nin:   source.Nin,
		Like:  convertContainsOperator(source.Contains),
		Nlike: convertContainsOperator(source.Ncontains),
		And:   convertQueryFilterStringList(source.And),
		Or:    convertQueryFilterStringList(source.Or),
	}
}

// ConvertQueryFilterStringPtr converts an API QueryFilterString pointer to internal filter.FilterString pointer.
func ConvertQueryFilterStringPtr(source *api.QueryFilterString) *filter.FilterString {
	if source == nil {
		return nil
	}

	return lo.ToPtr(ConvertQueryFilterString(*source))
}

// ConvertQueryFilterStringMapItem converts an API string query filter to internal filter.FilterString.
func ConvertQueryFilterStringMapItem(source api.QueryFilterStringMapItem) filter.FilterString {
	return filter.FilterString{
		Exists: source.Exists,
		Eq:     source.Eq,
		Ne:     source.Neq,
		In:     source.In,
		Nin:    source.Nin,
		Like:   convertContainsOperator(source.Contains),
		Nlike:  convertContainsOperator(source.Ncontains),
		And:    convertQueryFilterStringList(source.And),
		Or:     convertQueryFilterStringList(source.Or),
	}
}
