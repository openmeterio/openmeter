package plan

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripeTaxConfig(t *testing.T) {
	tests := []struct {
		Name          string
		TaxConfig     StripeTaxConfig
		ExpectedError error
	}{
		{
			Name: "valid",
			TaxConfig: StripeTaxConfig{
				Code: "txcd_99999999",
			},
			ExpectedError: nil,
		},
		{
			Name: "invalid",
			TaxConfig: StripeTaxConfig{
				Code: "invalid_tax_code",
			},
			ExpectedError: errors.New("invalid product tax code: invalid_tax_code"),
		},
	}

	for _, test := range tests {
		err := test.TaxConfig.Validate()
		assert.Equal(t, test.ExpectedError, err)
	}
}
