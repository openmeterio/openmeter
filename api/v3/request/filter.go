package request

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/samber/lo"
)

type Filter struct {
	// Equals
	Exists *bool `query:"exists"`
	// Equals operator
	Eq *string `query:"eq"`
	// Not equals operator
	Neq *string `query:"neq"`
	// Greater than operator
	Gt *string `query:"gt"`
	// Greater than or equal to operator
	Gte *string `query:"gte"`
	// Less than operator
	Lt *string `query:"lt"`
	// Less than or equal to operator
	Lte *string `query:"lte"`
	// Contains operator
	Contains *string `query:"contains"`
	// Or contains operator (in like)
	OrContains *[]string `query:"ocontains"`
	// Or equals operator (in)
	OrEq *[]string `query:"oeq"`
}

var ErrFilterMultipleFilterOperations = errors.New("only one filter operation is allowed")

func (f *Filter) Validate() error {
	nonNilFilters := lo.CountBy([]bool{
		f.Eq != nil, f.Neq != nil, f.Gt != nil, f.Gte != nil, f.Lt != nil, f.Lte != nil,
		f.Contains != nil, f.OrContains != nil, f.Exists != nil, f.OrEq != nil,
	}, func(b bool) bool { return b })
	if nonNilFilters > 1 {
		return ErrFilterMultipleFilterOperations
	}

	return nil
}

func (f *Filter) ToFilterString() *filter.FilterString {
	fs := &filter.FilterString{
		Eq:    f.Eq,
		Ne:    f.Neq,
		Gt:    f.Gt,
		Gte:   f.Gte,
		Lt:    f.Lt,
		Lte:   f.Lte,
		In:    f.OrEq,
		Ilike: f.Contains,
	}

	if f.Exists != nil {
		fs.Ne = lo.ToPtr("")
	}

	if f.OrContains != nil {
		fs.Or = lo.ToPtr(lo.Map(*f.OrContains, func(s string, _ int) filter.FilterString {
			return filter.FilterString{
				Ilike: &s,
			}
		}))
	}

	return fs
}
