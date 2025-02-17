package bloblang

import (
	"github.com/redpanda-data/benthos/v4/public/bloblang"
	"k8s.io/apimachinery/pkg/api/resource"
)

// init registers our custom function with Bloblang.
func init() {
	parseResourceSpec := bloblang.NewPluginSpec().
		Description("Parse a resource quantity from a string and convert it to decimal format.").
		Param(bloblang.NewStringParam("value").Description("The resource quantity to parse."))

	err := bloblang.RegisterFunctionV2("resource_quantity", parseResourceSpec, func(args *bloblang.ParsedParams) (bloblang.Function, error) {
		// Get the function arguments.
		value, err := args.GetString("value")
		if err != nil {
			return nil, err
		}

		if value == "" {
			return func() (any, error) {
				return 0, nil
			}, nil
		}

		// Parse the resource quantity.
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			return nil, err
		}

		// Return the function closure.
		return func() (any, error) {
			return quantity.AsDec().String(), nil
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
