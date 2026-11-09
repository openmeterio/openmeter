package billing

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestInvoicingConfigValidateSubscriptionEndProrationMode(t *testing.T) {
	tests := []struct {
		name        string
		mode        SubscriptionEndProrationMode
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing mode is invalid",
			mode:        "",
			wantErr:     true,
			errContains: "subscription end proration mode is required",
		},
		{
			name:    "bill actual period is valid",
			mode:    SubscriptionEndProrationModeBillActualPeriod,
			wantErr: false,
		},
		{
			name:    "bill full period is valid",
			mode:    SubscriptionEndProrationModeBillFullPeriod,
			wantErr: false,
		},
		{
			name:        "unknown mode is invalid",
			mode:        SubscriptionEndProrationMode("unknown"),
			wantErr:     true,
			errContains: "invalid subscription end proration mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultWorkflowConfig.Invoicing
			config.SubscriptionEndProrationMode = tt.mode

			err := config.Validate()
			if tt.wantErr {
				require.ErrorContains(t, err, tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func legacyTaxConfig(behavior *productcatalog.TaxBehavior, stripeCode string, taxCodeID *string) *productcatalog.TaxConfig {
	tc := &productcatalog.TaxConfig{
		Behavior:  behavior,
		TaxCodeID: taxCodeID,
	}
	if stripeCode != "" {
		tc.Stripe = &productcatalog.StripeTaxConfig{Code: stripeCode}
	}

	return tc
}

func TestWithDeprecatedDefaultTaxCodeEnforced(t *testing.T) {
	inclusive := lo.ToPtr(productcatalog.InclusiveTaxBehavior)
	exclusive := lo.ToPtr(productcatalog.ExclusiveTaxBehavior)

	tests := []struct {
		name                 string
		stored               *productcatalog.TaxConfig
		incoming             *productcatalog.TaxConfig
		wantErr              bool
		wantTaxCodeIDCleared bool
	}{
		// --- create (stored == nil) ---
		{
			name:     "create: taxCodeId set is rejected",
			stored:   nil,
			incoming: legacyTaxConfig(nil, "", lo.ToPtr("txcd_123")),
			wantErr:  true,
		},
		{
			name:     "create: stripe.code set is rejected",
			stored:   nil,
			incoming: legacyTaxConfig(nil, "txcd_123", nil),
			wantErr:  true,
		},
		{
			name:     "create: both stripe.code and taxCodeId set is rejected",
			stored:   nil,
			incoming: legacyTaxConfig(nil, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  true,
		},
		{
			name:     "create: only behavior set is allowed",
			stored:   nil,
			incoming: legacyTaxConfig(inclusive, "", nil),
			wantErr:  false,
		},
		{
			name:     "create: nil incoming is allowed",
			stored:   nil,
			incoming: nil,
			wantErr:  false,
		},
		{
			name:     "create: empty config is allowed",
			stored:   nil,
			incoming: legacyTaxConfig(nil, "", nil),
			wantErr:  false,
		},

		// --- update: adding a deprecated field where none existed ---
		{
			name:     "update: adding taxCodeId (stored had none) is rejected",
			stored:   legacyTaxConfig(inclusive, "", nil),
			incoming: legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_123")),
			wantErr:  true,
		},
		{
			name:     "update: adding stripe.code (stored had none) is rejected",
			stored:   legacyTaxConfig(inclusive, "", nil),
			incoming: legacyTaxConfig(inclusive, "txcd_123", nil),
			wantErr:  true,
		},

		// --- update: changing an existing deprecated value ---
		{
			name:     "update: changing taxCodeId value is rejected",
			stored:   legacyTaxConfig(nil, "", lo.ToPtr("txcd_old")),
			incoming: legacyTaxConfig(nil, "", lo.ToPtr("txcd_new")),
			wantErr:  true,
		},
		{
			name:     "update: changing stripe.code value is rejected",
			stored:   legacyTaxConfig(nil, "txcd_old", nil),
			incoming: legacyTaxConfig(nil, "txcd_new", nil),
			wantErr:  true,
		},

		// --- update: unchanged round-trip is allowed ---
		{
			name:     "update: unchanged taxCodeId + stripe.code round-trip is allowed",
			stored:   legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},

		// --- update: removal is allowed ---
		{
			name:     "update: full removal via nil incoming is allowed",
			stored:   legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: nil,
			wantErr:  false,
		},
		{
			name:     "update: full removal via all-nil fields is allowed",
			stored:   legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(nil, "", nil),
			wantErr:  false,
		},
		{
			name:                 "update: dropping stripe.code with unchanged taxCodeId removes the pair (taxCodeId cleared)",
			stored:               legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming:             legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			wantErr:              false,
			wantTaxCodeIDCleared: true,
		},
		{
			name:     "update: dropping taxCodeId with unchanged stripe.code is a legacy-client no-op echo (nothing cleared)",
			stored:   legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(inclusive, "txcd_123", nil),
			wantErr:  false,
		},
		{
			name:     "update: legacy stripe drop (taxCodeId is nil) is allowed",
			stored:   legacyTaxConfig(inclusive, "txcd_123", nil),
			incoming: legacyTaxConfig(inclusive, "", nil),
			wantErr:  false,
		},
		{
			name:     "update: taxCodeId drop (stripe is nil) is allowed",
			stored:   legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_123")),
			incoming: legacyTaxConfig(inclusive, "", nil),
			wantErr:  false,
		},

		// --- update: taxCodeId-only stored (no stripe block) ---
		{
			name:     "update: taxCodeId-only stored, faithful echo is a no-op (not a stripe removal; taxCodeId preserved)",
			stored:   legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},
		{
			name:     "update: taxCodeId-only stored, removal is allowed",
			stored:   legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(inclusive, "", nil),
			wantErr:  false,
		},
		{
			name:     "update: taxCodeId-only stored, adding stripe.code is rejected",
			stored:   legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  true,
		},
		{
			name:     "update: taxCodeId-only stored, changing taxCodeId is rejected",
			stored:   legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_other")),
			wantErr:  true,
		},

		// --- update: behavior is never restricted ---
		{
			name:     "update: adding behavior (no deprecated fields) is allowed",
			stored:   legacyTaxConfig(nil, "", nil),
			incoming: legacyTaxConfig(inclusive, "", nil),
			wantErr:  false,
		},
		{
			name:     "update: changing behavior with unchanged tax code is allowed",
			stored:   legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(exclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},
		{
			name:     "update: removing behavior with unchanged tax code is allowed",
			stored:   legacyTaxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: legacyTaxConfig(nil, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},

		// --- update: changing tax code while also changing behavior still rejected ---
		{
			name:     "update: changing taxCodeId alongside behavior is rejected",
			stored:   legacyTaxConfig(inclusive, "", lo.ToPtr("txcd_old")),
			incoming: legacyTaxConfig(exclusive, "", lo.ToPtr("txcd_new")),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var originalTaxCodeID *string
			if tt.incoming != nil {
				originalTaxCodeID = tt.incoming.TaxCodeID
			}

			result, err := WithDeprecatedDefaultTaxCodeEnforced(tt.incoming, tt.stored)
			if !tt.wantErr {
				require.NoError(t, err)

				if tt.incoming != nil {
					if tt.wantTaxCodeIDCleared {
						require.Nil(t, result.TaxCodeID)
					} else {
						require.Equal(t, originalTaxCodeID, result.TaxCodeID)
					}
				}

				return
			}

			require.Error(t, err)

			// The gate must surface as a billing.ValidationError so the HTTP encoders map it to 400.
			var validationErr ValidationError
			require.True(t, errors.As(err, &validationErr), "expected billing.ValidationError, got %T", err)
		})
	}
}

func invoicingConfigWithTaxCode(id string) InvoicingConfig {
	return InvoicingConfig{DefaultTaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: id}}
}

func TestInvoicingConfigValidateDefaultTaxCodeUnchanged(t *testing.T) {
	tests := []struct {
		name     string
		stored   InvoicingConfig
		incoming InvoicingConfig
		wantErr  bool
	}{
		{
			name:     "create: any tax code is rejected",
			stored:   InvoicingConfig{},
			incoming: invoicingConfigWithTaxCode("tc_1"),
			wantErr:  true,
		},
		{
			name:   "create: behavior only is allowed",
			stored: InvoicingConfig{},
			incoming: InvoicingConfig{DefaultTaxConfig: &productcatalog.TaxCodeConfig{
				Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			}},
		},
		{
			name:     "update: echo of the stored tax code is allowed",
			stored:   invoicingConfigWithTaxCode("tc_1"),
			incoming: invoicingConfigWithTaxCode("tc_1"),
		},
		{
			name:     "update: a different tax code is rejected",
			stored:   invoicingConfigWithTaxCode("tc_1"),
			incoming: invoicingConfigWithTaxCode("tc_2"),
			wantErr:  true,
		},
		{
			name:     "update: removal is allowed",
			stored:   invoicingConfigWithTaxCode("tc_1"),
			incoming: InvoicingConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.incoming.ValidateDefaultTaxCodeUnchanged(tt.stored)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			var validationErr ValidationError
			require.ErrorAs(t, err, &validationErr)
		})
	}
}
