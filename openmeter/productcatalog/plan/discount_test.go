package plan

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscount_JSON(t *testing.T) {
	tests := []struct {
		Name          string
		Discount      Discount
		ExpectedError bool
	}{
		{
			Name: "Valid",
			Discount: NewDiscountFrom(PercentageDiscount{
				Percentage: decimal.NewFromFloat(99.9),
				RateCards: []string{
					"ratecard-1",
					"ratecard-2",
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.Discount)
			require.NoError(t, err)

			t.Logf("Serialized Discount: %s", string(b))

			d := Discount{}
			err = json.Unmarshal(b, &d)
			require.NoError(t, err)

			assert.Equal(t, test.Discount, d)
		})
	}
}
