package request

import (
	"errors"

	"github.com/samber/lo"
)

type Filter struct {
	Eq        *string   `query:"eq"`
	Neq       *string   `query:"neq"`
	Gt        *string   `query:"gt"`
	Gte       *string   `query:"gte"`
	Lt        *string   `query:"lt"`
	Lte       *string   `query:"lte"`
	Contains  *[]string `query:"contains"`
	OContains *[]string `query:"ocontains"`
	Exists    *bool     `query:"exists"`
	OrEq      *string   `query:"oeq"`
}

var ErrFilterMultipleFilterOperations = errors.New("only one filter operation is allowed")

func (f *Filter) Validate() error {
	nonNilFilters := lo.CountBy([]bool{
		f.Eq != nil, f.Neq != nil, f.Gt != nil, f.Gte != nil, f.Lt != nil, f.Lte != nil,
		f.Contains != nil, f.OContains != nil, f.Exists != nil, f.OrEq != nil,
	}, func(b bool) bool { return b })
	if nonNilFilters > 1 {
		return ErrFilterMultipleFilterOperations
	}

	return nil
}
