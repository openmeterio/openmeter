//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./
package apiconverter

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./filter.gen.go
//
// The `goverter:ignore` directives below silence field-mismatch errors for
// fields that exist on the internal filter types but not on the v1 API types.
// If the v1 API (api/api.gen.go) is ever extended to expose those operators,
// remove the corresponding ignore entry so the generated converter copies
// them through — otherwise they will be silently dropped at the boundary.
var (
	// Exists/Contains/Ncontains are internal-only; the v1 FilterString has no
	// equivalent fields. Remove entries from the ignore list when v1 grows them.
	// goverter:ignore Exists Contains Ncontains
	ConvertString       func(api.FilterString) filter.FilterString
	ConvertStringPtr    func(*api.FilterString) *filter.FilterString
	ConvertStringMap    func(map[string]api.FilterString) map[string]filter.FilterString
	ConvertStringMapPtr func(*map[string]api.FilterString) *map[string]filter.FilterString
	// goverter:ignoreMissing
	ConvertIDExact    func(api.FilterIDExact) filter.FilterString
	ConvertIDExactPtr func(*api.FilterIDExact) *filter.FilterString
	ConvertInt        func(api.FilterInteger) filter.FilterInteger
	ConvertIntPtr     func(*api.FilterInteger) *filter.FilterInteger
	ConvertFloat      func(api.FilterFloat) filter.FilterFloat
	ConvertFloatPtr   func(*api.FilterFloat) *filter.FilterFloat
	// FilterTime.Eq is new on the internal type; v1 api.FilterTime does not
	// expose it. Remove this ignore when v1 grows an Eq field.
	// goverter:ignore Eq
	ConvertTime              func(api.FilterTime) filter.FilterTime
	ConvertTimePtr           func(*api.FilterTime) *filter.FilterTime
	ConvertBoolean           func(api.FilterBoolean) filter.FilterBoolean
	ConvertBooleanPtr        func(*api.FilterBoolean) *filter.FilterBoolean
	ConvertStringToAPI       func(*filter.FilterString) *api.FilterString
	ConvertStringMapToAPIPtr func(map[string]filter.FilterString) map[string]api.FilterString
)
