package productcatalog

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestTaxCodeConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *TaxCodeConfig
		wantErr string
	}{
		{
			name: "zero",
			cfg:  &TaxCodeConfig{},
		},
		{
			name: "behavior only",
			cfg:  &TaxCodeConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
		},
		{
			name:    "invalid behavior",
			cfg:     &TaxCodeConfig{Behavior: (*TaxBehavior)(lo.ToPtr("bad"))},
			wantErr: "validation error: invalid tax behavior: bad",
		},
		{
			name: "valid tax_code_id",
			cfg:  &TaxCodeConfig{TaxCodeID: "01AN4Z07BY79KA1307SR9X4MV3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

// TaxCodeConfig and the legacy TaxConfig share the same JSON keys, so either can read what the
// other wrote. That keeps older readers working when a field switches from one to the other.
func TestTaxCodeConfigJSONCompatibility(t *testing.T) {
	t.Run("zero config marshals to an empty object", func(t *testing.T) {
		got, err := json.Marshal(TaxCodeConfig{})
		require.NoError(t, err)

		assert.JSONEq(t, `{}`, string(got))
	})

	t.Run("empty object decodes into a legacy config without a tax code", func(t *testing.T) {
		var got TaxConfig
		require.NoError(t, json.Unmarshal([]byte(`{}`), &got))

		assert.Nil(t, got.TaxCodeID)
	})

	t.Run("full config round-trips through the legacy config", func(t *testing.T) {
		in := TaxCodeConfig{
			Behavior:  lo.ToPtr(ExclusiveTaxBehavior),
			TaxCodeID: "01AN4Z07BY79KA1307SR9X4MV3",
		}

		raw, err := json.Marshal(in)
		require.NoError(t, err)

		var legacy TaxConfig
		require.NoError(t, json.Unmarshal(raw, &legacy))
		assert.Equal(t, in.ToTaxConfig(), legacy)

		raw, err = json.Marshal(legacy)
		require.NoError(t, err)

		var out TaxCodeConfig
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, in, out)
	})
}

func TestTaxCodeConfigToTaxConfig(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		out := TaxCodeConfig{}.ToTaxConfig()
		assert.Nil(t, out.Behavior)
		assert.Nil(t, out.TaxCodeID)
	})

	t.Run("deep copy", func(t *testing.T) {
		behavior := InclusiveTaxBehavior
		id := "01AN4Z07BY79KA1307SR9X4MV3"
		c := TaxCodeConfig{Behavior: &behavior, TaxCodeID: id}
		out := c.ToTaxConfig()

		assert.Equal(t, behavior, *out.Behavior)
		assert.Equal(t, id, *out.TaxCodeID)

		// mutations to out must not alias back to c
		newBehavior := ExclusiveTaxBehavior
		out.Behavior = &newBehavior
		assert.Equal(t, InclusiveTaxBehavior, *c.Behavior, "ToTaxConfig must not share pointer fields")
	})
}

func TestTaxCodeConfigToLegacyTaxConfig(t *testing.T) {
	mapped := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-mapped"},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
	}
	unmapped := &taxcode.TaxCode{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-unmapped"}}
	taxCodes := map[string]*taxcode.TaxCode{mapped.ID: mapped, unmapped.ID: unmapped}

	tests := []struct {
		name string
		cfg  TaxCodeConfig
		want TaxConfig
	}{
		{
			name: "stripe code comes from the tax code's mapping",
			cfg:  TaxCodeConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior), TaxCodeID: mapped.ID},
			want: TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				Stripe:    &StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr(mapped.ID),
			},
		},
		{
			name: "tax code without a Stripe mapping",
			cfg:  TaxCodeConfig{TaxCodeID: unmapped.ID},
			want: TaxConfig{TaxCodeID: lo.ToPtr(unmapped.ID)},
		},
		{
			name: "tax code not among the resolved ones",
			cfg:  TaxCodeConfig{TaxCodeID: "tc-missing"},
			want: TaxConfig{TaxCodeID: lo.ToPtr("tc-missing")},
		},
		{
			name: "behavior only",
			cfg:  TaxCodeConfig{Behavior: lo.ToPtr(ExclusiveTaxBehavior)},
			want: TaxConfig{Behavior: lo.ToPtr(ExclusiveTaxBehavior)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.ToLegacyTaxConfig(taxCodes))
		})
	}
}

func TestTaxCodeConfigFrom(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, TaxCodeConfig{}, TaxCodeConfigFrom(nil))
	})

	t.Run("stripe-only config returns zero", func(t *testing.T) {
		cfg := &TaxConfig{Stripe: &StripeTaxConfig{Code: "txcd_10000000"}}
		assert.Equal(t, TaxCodeConfig{}, TaxCodeConfigFrom(cfg), "Stripe-only TaxConfig must yield zero TaxCodeConfig")
	})

	t.Run("deep copy", func(t *testing.T) {
		behavior := InclusiveTaxBehavior
		id := "01AN4Z07BY79KA1307SR9X4MV3"
		cfg := &TaxConfig{Behavior: &behavior, TaxCodeID: &id}
		out := TaxCodeConfigFrom(cfg)

		assert.Equal(t, behavior, *out.Behavior)
		assert.Equal(t, id, out.TaxCodeID)

		// mutations to out must not alias back to cfg
		newBehavior := ExclusiveTaxBehavior
		out.Behavior = &newBehavior
		assert.Equal(t, InclusiveTaxBehavior, *cfg.Behavior, "TaxCodeConfigFrom must not share pointer fields")
	})
}

func TestFromDBTaxColumns(t *testing.T) {
	const taxCodeID = "01JTAXC0DE0000000000000000"

	tests := []struct {
		name        string
		taxCodeID   *string
		taxBehavior *TaxBehavior
		want        *TaxCodeConfig
	}{
		{name: "no columns set", want: nil},
		{name: "tax code ID", taxCodeID: lo.ToPtr(taxCodeID), want: &TaxCodeConfig{TaxCodeID: taxCodeID}},
		{
			name:        "behavior only",
			taxBehavior: lo.ToPtr(InclusiveTaxBehavior),
			want:        &TaxCodeConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
		},
		{
			name:        "tax code ID and behavior",
			taxCodeID:   lo.ToPtr(taxCodeID),
			taxBehavior: lo.ToPtr(ExclusiveTaxBehavior),
			want:        &TaxCodeConfig{TaxCodeID: taxCodeID, Behavior: lo.ToPtr(ExclusiveTaxBehavior)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FromDBTaxColumns(tt.taxCodeID, tt.taxBehavior))
		})
	}
}

func TestTaxCodeConfigEqual(t *testing.T) {
	const id = "01JTAXC0DE0000000000000000"

	tests := []struct {
		name  string
		left  *TaxCodeConfig
		right *TaxCodeConfig
		want  bool
	}{
		{name: "both nil", want: true},
		{name: "nil and zero value", right: &TaxCodeConfig{}, want: true},
		{name: "zero value and nil", left: &TaxCodeConfig{}, want: true},
		{name: "nil and set", right: &TaxCodeConfig{TaxCodeID: id}},
		{
			name:  "same values in different pointers",
			left:  &TaxCodeConfig{TaxCodeID: id, Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			right: &TaxCodeConfig{TaxCodeID: id, Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			want:  true,
		},
		{
			name:  "different tax code ID",
			left:  &TaxCodeConfig{TaxCodeID: id},
			right: &TaxCodeConfig{TaxCodeID: "01JTAXC0DE0000000000000001"},
		},
		{
			name:  "different behavior",
			left:  &TaxCodeConfig{TaxCodeID: id, Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			right: &TaxCodeConfig{TaxCodeID: id, Behavior: lo.ToPtr(ExclusiveTaxBehavior)},
		},
		{
			name:  "behavior set on one side only",
			left:  &TaxCodeConfig{TaxCodeID: id, Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			right: &TaxCodeConfig{TaxCodeID: id},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.left.Equal(tt.right))
			assert.Equal(t, tt.want, tt.right.Equal(tt.left))
		})
	}
}

func TestTaxCodeConfigClone(t *testing.T) {
	original := TaxCodeConfig{
		TaxCodeID: "01JTAXC0DE0000000000000000",
		Behavior:  lo.ToPtr(InclusiveTaxBehavior),
	}

	cloned := original.Clone()
	require.True(t, original.Equal(&cloned))
	require.NotSame(t, original.Behavior, cloned.Behavior)

	*cloned.Behavior = ExclusiveTaxBehavior
	assert.Equal(t, InclusiveTaxBehavior, *original.Behavior)

	assert.Equal(t, TaxCodeConfig{}, TaxCodeConfig{}.Clone())
}

func TestTaxConfigValidation(t *testing.T) {
	tests := []struct {
		Name          string
		TaxConfig     TaxConfig
		ExpectedError error
	}{
		{
			Name: "stripe valid",
			TaxConfig: TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			ExpectedError: nil,
		},
		{
			Name: "stripe invalid",
			TaxConfig: TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "invalid_tax_code",
				},
			},
			ExpectedError: errors.New("validation error: invalid stripe config: validation error: invalid product tax code: invalid_tax_code"),
		},
		{
			Name: "behavior valid",
			TaxConfig: TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			ExpectedError: nil,
		},
		{
			Name: "behavior invalid",
			TaxConfig: TaxConfig{
				Behavior: (*TaxBehavior)(lo.ToPtr("invalid_behavior")),
			},
			ExpectedError: errors.New("validation error: invalid tax behavior: invalid_behavior"),
		},
	}

	for _, test := range tests {
		err := test.TaxConfig.Validate()
		if test.ExpectedError == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, test.ExpectedError.Error())
		}
	}
}

func TestTaxConfigEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  *TaxConfig
		Right *TaxConfig

		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: nil,
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: nil,
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
			ExpectedResult: false,
		},
		{
			Name: "Equal - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			Right:          &TaxConfig{},
			ExpectedResult: false,
		},
		{
			Name: "Right diff - behavior",
			Left: nil,
			Right: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
			},
			ExpectedResult: false,
		},
		{
			Name: "Equal - TaxCodeID",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			Right: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff - TaxCodeID",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			Right:          &TaxConfig{},
			ExpectedResult: false,
		},
		{
			Name: "Right diff - TaxCodeID",
			Left: nil,
			Right: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff - TaxCodeID",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			Right: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
			},
			ExpectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			match := test.Left.Equal(test.Right)
			assert.Equal(t, test.ExpectedResult, match)
		})
	}
}

func TestTaxConfigClone(t *testing.T) {
	original := TaxConfig{
		Behavior:  lo.ToPtr(InclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
		Stripe: &StripeTaxConfig{
			Code: "txcd_99999999",
		},
	}

	cloned := original.Clone()

	// Cloned value must be equal to the original
	assert.True(t, original.Equal(&cloned))

	// Mutate the clone's pointer fields
	*cloned.TaxCodeID = "01AN4Z07BY79KA1307SR9X4MV4"
	*cloned.Behavior = ExclusiveTaxBehavior
	cloned.Stripe.Code = "txcd_00000000"

	// Original must be unchanged
	assert.Equal(t, "01AN4Z07BY79KA1307SR9X4MV3", *original.TaxCodeID)
	assert.Equal(t, InclusiveTaxBehavior, *original.Behavior)
	assert.Equal(t, "txcd_99999999", original.Stripe.Code)

	// Pointers must not be shared
	assert.NotSame(t, original.TaxCodeID, cloned.TaxCodeID)
	assert.NotSame(t, original.Behavior, cloned.Behavior)
	assert.NotSame(t, original.Stripe, cloned.Stripe)

	// Values must now differ
	assert.False(t, original.Equal(&cloned))
}
