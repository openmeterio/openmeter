package billing

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestCheckProfileTaxConfigDeprecation(t *testing.T) {
	taxConfig := func(behavior *productcatalog.TaxBehavior, stripeCode string, taxCodeID *string) *productcatalog.TaxConfig {
		tc := &productcatalog.TaxConfig{
			Behavior:  behavior,
			TaxCodeID: taxCodeID,
		}
		if stripeCode != "" {
			tc.Stripe = &productcatalog.StripeTaxConfig{Code: stripeCode}
		}
		return tc
	}

	inclusive := lo.ToPtr(productcatalog.InclusiveTaxBehavior)
	exclusive := lo.ToPtr(productcatalog.ExclusiveTaxBehavior)

	tests := []struct {
		name     string
		stored   *productcatalog.TaxConfig
		incoming *productcatalog.TaxConfig
		wantErr  bool
	}{
		// --- create (stored == nil) ---
		{
			name:     "create: taxCodeId set is rejected",
			stored:   nil,
			incoming: taxConfig(nil, "", lo.ToPtr("txcd_123")),
			wantErr:  true,
		},
		{
			name:     "create: stripe.code set is rejected",
			stored:   nil,
			incoming: taxConfig(nil, "txcd_123", nil),
			wantErr:  true,
		},
		{
			name:     "create: only behavior set is allowed",
			stored:   nil,
			incoming: taxConfig(inclusive, "", nil),
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
			incoming: taxConfig(nil, "", nil),
			wantErr:  false,
		},

		// --- update: adding a deprecated field where none existed ---
		{
			name:     "update: adding taxCodeId (stored had none) is rejected",
			stored:   taxConfig(inclusive, "", nil),
			incoming: taxConfig(inclusive, "", lo.ToPtr("txcd_123")),
			wantErr:  true,
		},
		{
			name:     "update: adding stripe.code (stored had none) is rejected",
			stored:   taxConfig(inclusive, "", nil),
			incoming: taxConfig(inclusive, "txcd_123", nil),
			wantErr:  true,
		},

		// --- update: changing an existing deprecated value ---
		{
			name:     "update: changing taxCodeId value is rejected",
			stored:   taxConfig(nil, "", lo.ToPtr("txcd_old")),
			incoming: taxConfig(nil, "", lo.ToPtr("txcd_new")),
			wantErr:  true,
		},
		{
			name:     "update: changing stripe.code value is rejected",
			stored:   taxConfig(nil, "txcd_old", nil),
			incoming: taxConfig(nil, "txcd_new", nil),
			wantErr:  true,
		},

		// --- update: unchanged round-trip is allowed ---
		{
			name:     "update: unchanged taxCodeId + stripe.code round-trip is allowed",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},

		// --- update: removal is allowed ---
		{
			name:     "update: full removal via nil incoming is allowed",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: nil,
			wantErr:  false,
		},
		{
			name:     "update: full removal via all-nil fields is allowed",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: taxConfig(nil, "", nil),
			wantErr:  false,
		},
		{
			name:     "update: partial removal (drop stripe, keep taxCodeId) is rejected",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: taxConfig(inclusive, "", lo.ToPtr("txcd_abc")),
			wantErr:  true,
		},
		{
			name:     "update: partial removal (drop taxCodeId, keep stripe) is rejected",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: taxConfig(inclusive, "txcd_123", nil),
			wantErr:  true,
		},
		{
			name:     "update: legacy stripe drop (taxCodeId is nil) is allowed",
			stored:   taxConfig(inclusive, "txcd_123", nil),
			incoming: taxConfig(inclusive, "", nil),
			wantErr:  false,
		},
		{
			name:     "update: taxCodeId drop (stripe is nil) is allowed",
			stored:   taxConfig(inclusive, "", lo.ToPtr("txcd_123")),
			incoming: taxConfig(inclusive, "", nil),
			wantErr:  false,
		},

		// --- update: behavior is never restricted ---
		{
			name:     "update: adding behavior (no deprecated fields) is allowed",
			stored:   taxConfig(nil, "", nil),
			incoming: taxConfig(inclusive, "", nil),
			wantErr:  false,
		},
		{
			name:     "update: changing behavior with unchanged tax code is allowed",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: taxConfig(exclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},
		{
			name:     "update: removing behavior with unchanged tax code is allowed",
			stored:   taxConfig(inclusive, "txcd_123", lo.ToPtr("txcd_abc")),
			incoming: taxConfig(nil, "txcd_123", lo.ToPtr("txcd_abc")),
			wantErr:  false,
		},

		// --- update: changing tax code while also changing behavior still rejected ---
		{
			name:     "update: changing taxCodeId alongside behavior is rejected",
			stored:   taxConfig(inclusive, "", lo.ToPtr("txcd_old")),
			incoming: taxConfig(exclusive, "", lo.ToPtr("txcd_new")),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckProfileTaxConfigDeprecation(tt.stored, tt.incoming)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			// The gate must surface as a billing.ValidationError so the HTTP encoders map it to 400.
			var validationErr ValidationError
			require.True(t, errors.As(err, &validationErr), "expected billing.ValidationError, got %T", err)
		})
	}
}
