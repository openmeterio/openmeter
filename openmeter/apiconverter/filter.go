//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./
package apiconverter

import (
	"github.com/openmeterio/openmeter/api/models"
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
	ConvertString       func(models.FilterString) filter.FilterString
	ConvertStringPtr    func(*models.FilterString) *filter.FilterString
	ConvertStringMap    func(map[string]models.FilterString) map[string]filter.FilterString
	ConvertStringMapPtr func(*map[string]models.FilterString) *map[string]filter.FilterString
	// goverter:ignoreMissing
	ConvertIDExact    func(models.FilterIDExact) filter.FilterString
	ConvertIDExactPtr func(*models.FilterIDExact) *filter.FilterString
	ConvertInt        func(models.FilterInteger) filter.FilterInteger
	ConvertIntPtr     func(*models.FilterInteger) *filter.FilterInteger
	ConvertFloat      func(models.FilterFloat) filter.FilterFloat
	ConvertFloatPtr   func(*models.FilterFloat) *filter.FilterFloat
	// FilterTime.Eq is new on the internal type; v1 models.FilterTime does not
	// expose it. Remove this ignore when v1 grows an Eq field.
	// goverter:ignore Eq
	// goverter:ignore Exists
	ConvertTime              func(models.FilterTime) filter.FilterTime
	ConvertTimePtr           func(*models.FilterTime) *filter.FilterTime
	ConvertBoolean           func(models.FilterBoolean) filter.FilterBoolean
	ConvertBooleanPtr        func(*models.FilterBoolean) *filter.FilterBoolean
	ConvertStringToAPI       func(*filter.FilterString) *models.FilterString
	ConvertStringMapToAPIPtr func(map[string]filter.FilterString) map[string]models.FilterString
)
