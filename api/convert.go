//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./

package api

// This file contains the conversion functions for the API types.
// This can be used to convert between similar API types, as the oapi-codegen generates
// different types for the same Go struct.

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:enum no
var (
	FromBillingDiscountPercentageToDiscountPercentage func(BillingDiscountPercentage) DiscountPercentage
	FromBillingDiscountUsageToDiscountUsage           func(BillingDiscountUsage) DiscountUsage
)
