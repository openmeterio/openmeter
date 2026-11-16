//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./
package apiconverter

import (
	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/pkg/filter"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./filter.gen.go
//
// The `goverter:ignore` directives below silence field-mismatch errors for
// fields that exist on the internal filter types but not on the v1 API types.
// If the v1 API (api/models/models.gen.go) is ever extended to expose those operators,
// remove the corresponding ignore entry so the generated converter copies
// them through — otherwise they will be silently dropped at the boundary.
var (
	// Exists/Contains/Ncontains are internal-only; the v1 FilterString has no
	// equivalent fields. Remove entries from the ignore list when v1 grows them.
	// goverter:ignore Exists Contains Ncontains
	ConvertString       func(apiv1models.FilterString) filter.FilterString
	ConvertStringPtr    func(*apiv1models.FilterString) *filter.FilterString
	ConvertStringMap    func(map[string]apiv1models.FilterString) map[string]filter.FilterString
	ConvertStringMapPtr func(*map[string]apiv1models.FilterString) *map[string]filter.FilterString
	// goverter:ignoreMissing
	ConvertIDExact    func(apiv1models.FilterIDExact) filter.FilterString
	ConvertIDExactPtr func(*apiv1models.FilterIDExact) *filter.FilterString
	ConvertInt        func(apiv1models.FilterInteger) filter.FilterInteger
	ConvertIntPtr     func(*apiv1models.FilterInteger) *filter.FilterInteger
	ConvertFloat      func(apiv1models.FilterFloat) filter.FilterFloat
	ConvertFloatPtr   func(*apiv1models.FilterFloat) *filter.FilterFloat
	// FilterTime.Eq is new on the internal type; apiv1models.FilterTime does not
	// expose it. Remove this ignore when v1 grows an Eq field.
	// goverter:ignore Eq
	// goverter:ignore Exists
	ConvertTime              func(apiv1models.FilterTime) filter.FilterTime
	ConvertTimePtr           func(*apiv1models.FilterTime) *filter.FilterTime
	ConvertBoolean           func(apiv1models.FilterBoolean) filter.FilterBoolean
	ConvertBooleanPtr        func(*apiv1models.FilterBoolean) *filter.FilterBoolean
	ConvertStringToAPI       func(*filter.FilterString) *apiv1models.FilterString
	ConvertStringMapToAPIPtr func(map[string]filter.FilterString) map[string]apiv1models.FilterString
)
