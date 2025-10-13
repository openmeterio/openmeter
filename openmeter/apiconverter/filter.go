//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./
package apiconverter

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./filter.gen.go
var (
	ConvertString       func(api.FilterString) filter.FilterString
	ConvertStringPtr    func(*api.FilterString) *filter.FilterString
	ConvertStringMap    func(map[string]api.FilterString) map[string]filter.FilterString
	ConvertStringMapPtr func(*map[string]api.FilterString) *map[string]filter.FilterString
	// goverter:ignoreMissing
	ConvertIDExact           func(api.FilterIDExact) filter.FilterString
	ConvertIDExactPtr        func(*api.FilterIDExact) *filter.FilterString
	ConvertInt               func(api.FilterInteger) filter.FilterInteger
	ConvertIntPtr            func(*api.FilterInteger) *filter.FilterInteger
	ConvertFloat             func(api.FilterFloat) filter.FilterFloat
	ConvertFloatPtr          func(*api.FilterFloat) *filter.FilterFloat
	ConvertTime              func(api.FilterTime) filter.FilterTime
	ConvertTimePtr           func(*api.FilterTime) *filter.FilterTime
	ConvertBoolean           func(api.FilterBoolean) filter.FilterBoolean
	ConvertBooleanPtr        func(*api.FilterBoolean) *filter.FilterBoolean
	ConvertStringToAPI       func(*filter.FilterString) *api.FilterString
	ConvertStringMapToAPIPtr func(map[string]filter.FilterString) map[string]api.FilterString
)
