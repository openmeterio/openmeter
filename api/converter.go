//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package api

import (
	"github.com/openmeterio/openmeter/pkg/filter"
)

// goverter:converter
// goverter:skipCopySameType
// goverter:output:file ./converter.gen.go
type Converter interface {
	ConvertFilterString(*filter.FilterString) *FilterString
	ConvertFilterInteger(*filter.FilterInteger) *FilterInteger
	ConvertFilterFloat(*filter.FilterFloat) *FilterFloat
	ConvertFilterTime(*filter.FilterTime) *FilterTime
}
