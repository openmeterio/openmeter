package productcatalog

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestTaxCodeConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *TaxCodeConfig
		wantErr string
	}{
		{
			name:    "nil",
			cfg:     nil,
			wantErr: "",
		},
		{
			name:    "valid behavior",
			cfg:     &TaxCodeConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			wantErr: "",
		},
		{
			name:    "invalid behavior",
			cfg:     &TaxCodeConfig{Behavior: (*TaxBehavior)(lo.ToPtr("bad"))},
			wantErr: "validation error: invalid tax behavior: bad",
		},
		{
			name:    "valid tax_code_id",
			cfg:     &TaxCodeConfig{TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3")},
			wantErr: "",
		},
		{
			name:    "empty tax_code_id",
			cfg:     &TaxCodeConfig{TaxCodeID: lo.ToPtr("")},
			wantErr: "validation error: tax_code_id must not be empty when set",
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

func TestTaxCodeConfigToTaxConfig(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *TaxCodeConfig
		assert.Nil(t, c.ToTaxConfig())
	})

	t.Run("deep copy", func(t *testing.T) {
		behavior := InclusiveTaxBehavior
		id := "01AN4Z07BY79KA1307SR9X4MV3"
		c := &TaxCodeConfig{Behavior: &behavior, TaxCodeID: &id}
		out := c.ToTaxConfig()

		require.NotNil(t, out)
		assert.Equal(t, behavior, *out.Behavior)
		assert.Equal(t, id, *out.TaxCodeID)

		// mutations to out must not alias back to c
		newBehavior := ExclusiveTaxBehavior
		out.Behavior = &newBehavior
		assert.Equal(t, InclusiveTaxBehavior, *c.Behavior, "ToTaxConfig must not share pointer fields")
	})
}

func TestTaxCodeConfigFrom(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, TaxCodeConfigFrom(nil))
	})

	t.Run("stripe-only config returns nil", func(t *testing.T) {
		cfg := &TaxConfig{Stripe: &StripeTaxConfig{Code: "txcd_10000000"}}
		assert.Nil(t, TaxCodeConfigFrom(cfg), "Stripe-only TaxConfig must yield nil TaxCodeConfig")
	})

	t.Run("deep copy", func(t *testing.T) {
		behavior := InclusiveTaxBehavior
		id := "01AN4Z07BY79KA1307SR9X4MV3"
		cfg := &TaxConfig{Behavior: &behavior, TaxCodeID: &id}
		out := TaxCodeConfigFrom(cfg)

		require.NotNil(t, out)
		assert.Equal(t, behavior, *out.Behavior)
		assert.Equal(t, id, *out.TaxCodeID)

		// mutations to out must not alias back to cfg
		newBehavior := ExclusiveTaxBehavior
		out.Behavior = &newBehavior
		assert.Equal(t, InclusiveTaxBehavior, *cfg.Behavior, "TaxCodeConfigFrom must not share pointer fields")
	})
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

func TestMergeTaxConfigs(t *testing.T) {
	tests := []struct {
		Name     string
		Left     *TaxConfig
		Right    *TaxConfig
		Expected *TaxConfig
	}{
		{
			Name: "Left nil",
			Left: nil,
			Right: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name: "Right nil",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: nil,
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name:     "Left and Right nil",
			Left:     nil,
			Right:    nil,
			Expected: nil,
		},
		{
			Name: "Right overrides left fully",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
		},
		{
			Name: "Right overrides left partially - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name: "Right overrides left partially - stripe",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
		},
		{
			Name: "Right overrides left partially - TaxCodeID",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name: "Right overrides left partially - TaxCodeID and Stripe",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
		},
		{
			Name: "TaxCode resolved entity is dropped — merge is intent-level only",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
				TaxCode:   &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01AN4Z07BY79KA1307SR9X4MV3"}},
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
			},
			Expected: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
				Behavior:  lo.ToPtr(ExclusiveTaxBehavior),
				// TaxCode intentionally omitted from merge result
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			merged := MergeTaxConfigs(test.Left, test.Right)
			assert.Equal(t, test.Expected, merged)

			// Single-non-nil inputs must return an independent copy, not the original pointer.
			if test.Left == nil && test.Right != nil {
				assert.NotSame(t, test.Right, merged, "MergeTaxConfigs with nil base must return independent copy")
			}
			if test.Right == nil && test.Left != nil {
				assert.NotSame(t, test.Left, merged, "MergeTaxConfigs with nil overrides must return independent copy")
			}
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

func TestTaxConfigCloneWithTaxCode(t *testing.T) {
	desc := "Software - SaaS"
	original := TaxConfig{
		Stripe: &StripeTaxConfig{Code: "txcd_10000000"},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-1"},
			Description:  &desc,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
			},
		},
	}

	cloned := original.Clone()

	// TaxCode pointer must not be shared
	assert.NotSame(t, original.TaxCode, cloned.TaxCode)

	// Mutating clone's AppMappings must not affect original
	cloned.TaxCode.AppMappings = append(cloned.TaxCode.AppMappings, taxcode.TaxCodeAppMapping{
		AppType: app.AppTypeStripe, TaxCode: "txcd_99999999",
	})
	assert.Len(t, original.TaxCode.AppMappings, 1, "original AppMappings slice must not grow")

	// Mutating clone's Description must not affect original
	*cloned.TaxCode.Description = "mutated"
	assert.Equal(t, "Software - SaaS", *original.TaxCode.Description)

	// Clone of config with nil TaxCode must have nil TaxCode
	nilTCConfig := TaxConfig{Stripe: &StripeTaxConfig{Code: "txcd_10000000"}}
	clonedNil := nilTCConfig.Clone()
	assert.Nil(t, clonedNil.TaxCode)
}

func TestBackfillTaxConfig(t *testing.T) {
	newTaxCode := func(id, stripeCode string) *taxcode.TaxCode {
		tc := &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: id},
		}
		if stripeCode != "" {
			tc.AppMappings = taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: stripeCode},
			}
		}
		return tc
	}

	tests := []struct {
		name        string
		cfg         *TaxConfig
		taxBehavior *TaxBehavior
		tc          *taxcode.TaxCode
		want        *TaxConfig
	}{
		{
			name:        "all nil returns nil",
			cfg:         nil,
			taxBehavior: nil,
			tc:          nil,
			want:        nil,
		},
		{
			name:        "tc with no stripe mapping is no-op",
			cfg:         nil,
			taxBehavior: nil,
			tc:          newTaxCode("tc-1", ""),
			want:        nil,
		},
		{
			name:        "behavior only fills empty config",
			cfg:         nil,
			taxBehavior: lo.ToPtr(ExclusiveTaxBehavior),
			tc:          nil,
			want:        &TaxConfig{Behavior: lo.ToPtr(ExclusiveTaxBehavior)},
		},
		{
			name:        "tc with stripe mapping fills empty config",
			cfg:         nil,
			taxBehavior: nil,
			tc:          newTaxCode("tc-1", "txcd_10000000"),
			want: &TaxConfig{
				Stripe:    &StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr("tc-1"),
			},
		},
		{
			name:        "both behavior and tc fill empty config",
			cfg:         nil,
			taxBehavior: lo.ToPtr(InclusiveTaxBehavior),
			tc:          newTaxCode("tc-1", "txcd_10000000"),
			want: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				Stripe:    &StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr("tc-1"),
			},
		},
		{
			name:        "existing stripe not overwritten by tc",
			cfg:         &TaxConfig{Stripe: &StripeTaxConfig{Code: "txcd_20000000"}},
			taxBehavior: nil,
			tc:          newTaxCode("tc-1", "txcd_10000000"),
			want: &TaxConfig{
				Stripe:    &StripeTaxConfig{Code: "txcd_20000000"},
				TaxCodeID: lo.ToPtr("tc-1"),
			},
		},
		{
			name:        "existing behavior not overwritten by taxBehavior",
			cfg:         &TaxConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			taxBehavior: lo.ToPtr(ExclusiveTaxBehavior),
			tc:          nil,
			want:        &TaxConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
		},
		{
			name:        "existing TaxCodeID not overwritten by tc",
			cfg:         &TaxConfig{TaxCodeID: lo.ToPtr("existing-id")},
			taxBehavior: nil,
			tc:          newTaxCode("new-id", "txcd_10000000"),
			want: &TaxConfig{
				TaxCodeID: lo.ToPtr("existing-id"),
				Stripe:    &StripeTaxConfig{Code: "txcd_10000000"},
			},
		},
		{
			name: "fully populated cfg is untouched",
			cfg: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				Stripe:    &StripeTaxConfig{Code: "txcd_20000000"},
				TaxCodeID: lo.ToPtr("existing-id"),
			},
			taxBehavior: lo.ToPtr(ExclusiveTaxBehavior),
			tc:          newTaxCode("new-id", "txcd_10000000"),
			want: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				Stripe:    &StripeTaxConfig{Code: "txcd_20000000"},
				TaxCodeID: lo.ToPtr("existing-id"),
			},
		},
		{
			name:        "tc with empty ID does not set TaxCodeID",
			cfg:         nil,
			taxBehavior: nil,
			tc:          newTaxCode("", "txcd_10000000"),
			want: &TaxConfig{
				Stripe: &StripeTaxConfig{Code: "txcd_10000000"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BackfillTaxConfig(tt.cfg, tt.taxBehavior, tt.tc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBackfillTaxConfigPointerIsolation(t *testing.T) {
	t.Run("filled Behavior is independent from taxBehavior arg", func(t *testing.T) {
		behavior := ExclusiveTaxBehavior
		got := BackfillTaxConfig(nil, &behavior, nil)
		require.NotNil(t, got)
		require.NotNil(t, got.Behavior)

		// mutate source — must not affect result
		behavior = InclusiveTaxBehavior
		assert.Equal(t, ExclusiveTaxBehavior, *got.Behavior, "BackfillTaxConfig must deep-copy taxBehavior")
	})

	t.Run("filled TaxCodeID is independent from TaxCode entity ID field", func(t *testing.T) {
		tc := &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "original-id"},
			AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
		}
		got := BackfillTaxConfig(nil, nil, tc)
		require.NotNil(t, got)
		require.NotNil(t, got.TaxCodeID)

		// mutate source — must not affect result
		tc.ID = "mutated-id"
		assert.Equal(t, "original-id", *got.TaxCodeID, "BackfillTaxConfig must deep-copy TaxCode ID")
	})
}

// stubTaxCodeService is a minimal taxcode.Service implementation for unit tests.
// Only GetTaxCode and GetOrCreateByAppMapping are wired; other methods panic.
type stubTaxCodeService struct {
	getTaxCode              func(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error)
	getOrCreateByAppMapping func(ctx context.Context, input taxcode.GetOrCreateByAppMappingInput) (taxcode.TaxCode, error)
}

func (s *stubTaxCodeService) GetTaxCode(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	return s.getTaxCode(ctx, input)
}

func (s *stubTaxCodeService) GetOrCreateByAppMapping(ctx context.Context, input taxcode.GetOrCreateByAppMappingInput) (taxcode.TaxCode, error) {
	return s.getOrCreateByAppMapping(ctx, input)
}

func (s *stubTaxCodeService) CreateTaxCode(_ context.Context, _ taxcode.CreateTaxCodeInput) (taxcode.TaxCode, error) {
	panic("not implemented")
}

func (s *stubTaxCodeService) UpdateTaxCode(_ context.Context, _ taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
	panic("not implemented")
}

func (s *stubTaxCodeService) ListTaxCodes(_ context.Context, _ taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	panic("not implemented")
}

func (s *stubTaxCodeService) GetTaxCodeByAppMapping(_ context.Context, _ taxcode.GetTaxCodeByAppMappingInput) (taxcode.TaxCode, error) {
	panic("not implemented")
}

func (s *stubTaxCodeService) DeleteTaxCode(_ context.Context, _ taxcode.DeleteTaxCodeInput) error {
	panic("not implemented")
}

func TestResolveTaxConfig(t *testing.T) {
	const ns = "test-ns"

	tcWithStripe := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: "tc-stripe"},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
	}
	tcWithoutStripe := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: "tc-no-stripe"},
		AppMappings:  taxcode.TaxCodeAppMappings{},
	}

	tests := []struct {
		name    string
		svc     taxcode.Service
		cfg     *TaxConfig
		wantCfg *TaxConfig
		wantErr string
	}{
		{
			name:    "nil cfg is no-op",
			cfg:     nil,
			wantCfg: nil,
		},
		{
			name:    "nil svc with non-nil cfg returns error",
			svc:     nil,
			cfg:     &TaxConfig{TaxCodeID: lo.ToPtr("tc-stripe")},
			wantErr: "taxcode service is required",
		},
		{
			name: "TaxCodeID set, entity has Stripe mapping — Stripe backfilled",
			svc: &stubTaxCodeService{
				getTaxCode: func(_ context.Context, _ taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
					return tcWithStripe, nil
				},
			},
			cfg:     &TaxConfig{TaxCodeID: lo.ToPtr("tc-stripe")},
			wantCfg: &TaxConfig{TaxCodeID: lo.ToPtr("tc-stripe"), Stripe: &StripeTaxConfig{Code: "txcd_10000000"}},
		},
		{
			name: "TaxCodeID set, entity has no Stripe mapping — Stripe cleared",
			svc: &stubTaxCodeService{
				getTaxCode: func(_ context.Context, _ taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
					return tcWithoutStripe, nil
				},
			},
			cfg:     &TaxConfig{TaxCodeID: lo.ToPtr("tc-no-stripe"), Stripe: &StripeTaxConfig{Code: "txcd_10000000"}},
			wantCfg: &TaxConfig{TaxCodeID: lo.ToPtr("tc-no-stripe"), Stripe: nil},
		},
		{
			name: "both TaxCodeID and Stripe code set — TaxCodeID wins, Stripe overwritten from entity",
			svc: &stubTaxCodeService{
				getTaxCode: func(_ context.Context, _ taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
					return tcWithStripe, nil
				},
			},
			cfg:     &TaxConfig{TaxCodeID: lo.ToPtr("tc-stripe"), Stripe: &StripeTaxConfig{Code: "txcd_99999999"}},
			wantCfg: &TaxConfig{TaxCodeID: lo.ToPtr("tc-stripe"), Stripe: &StripeTaxConfig{Code: "txcd_10000000"}},
		},
		{
			name: "TaxCodeID set, entity not found — validation error",
			svc: &stubTaxCodeService{
				getTaxCode: func(_ context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
					return taxcode.TaxCode{}, taxcode.NewTaxCodeNotFoundError(input.ID)
				},
			},
			cfg:     &TaxConfig{TaxCodeID: lo.ToPtr("missing-id")},
			wantErr: "validation error: tax code missing-id not found",
		},
		{
			name: "Stripe code set — TaxCodeID stamped",
			svc: &stubTaxCodeService{
				getOrCreateByAppMapping: func(_ context.Context, _ taxcode.GetOrCreateByAppMappingInput) (taxcode.TaxCode, error) {
					return tcWithStripe, nil
				},
			},
			cfg:     &TaxConfig{Stripe: &StripeTaxConfig{Code: "txcd_10000000"}},
			wantCfg: &TaxConfig{Stripe: &StripeTaxConfig{Code: "txcd_10000000"}, TaxCodeID: lo.ToPtr("tc-stripe")},
		},
		{
			name:    "neither TaxCodeID nor Stripe code — no-op",
			svc:     &stubTaxCodeService{},
			cfg:     &TaxConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
			wantCfg: &TaxConfig{Behavior: lo.ToPtr(InclusiveTaxBehavior)},
		},
		{
			name:    "empty Stripe code — no-op",
			svc:     &stubTaxCodeService{},
			cfg:     &TaxConfig{Stripe: &StripeTaxConfig{Code: ""}},
			wantCfg: &TaxConfig{Stripe: &StripeTaxConfig{Code: ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ResolveTaxConfig(t.Context(), tt.svc, ns, tt.cfg)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCfg, tt.cfg)
		})
	}
}
