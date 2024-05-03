//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml
package api

const (
	// DefaultCreditQueryLimit specifies how many entries to return by default for credit related queries
	DefaultCreditsQueryLimit = 1000
)
