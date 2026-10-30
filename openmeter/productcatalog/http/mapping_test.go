package http_test

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// stubBackfillResolver returns a Tax Code with a fixed ID (or err) and records the inputs it got.
type stubBackfillResolver struct {
	id    string
	err   error
	calls []productcatalog.ResolveOrBackfillTaxCodeInput
}

func (s *stubBackfillResolver) ResolveOrBackfillTaxCode(_ context.Context, input productcatalog.ResolveOrBackfillTaxCodeInput) (*taxcode.TaxCode, error) {
	s.calls = append(s.calls, input)

	if s.err != nil {
		return nil, s.err
	}

	return &taxcode.TaxCode{NamespacedID: models.NamespacedID{Namespace: input.Namespace, ID: s.id}}, nil
}

func TestAsTaxCodeConfig(t *testing.T) {
	const (
		namespace  = "ns"
		inputID    = "01JTAXCODE00000000000000000"
		resolvedID = "01JTAXCODE00000000000000001"
		stripeCode = "txcd_10000000"
	)

	resolverErr := errors.New("db down")

	tests := []struct {
		name        string
		in          api.TaxConfig
		resolverErr error
		want        *productcatalog.TaxCodeConfig
		wantCalls   []productcatalog.ResolveOrBackfillTaxCodeInput
		wantErr     func(t *testing.T, err error)
	}{
		{
			name: "empty input",
			in:   api.TaxConfig{},
		},
		{
			name: "empty stripe code",
			in:   api.TaxConfig{Stripe: &api.StripeTaxConfig{}},
		},
		{
			name: "behavior only",
			in:   api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorInclusive)},
			want: &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)},
		},
		{
			name: "tax code ID wins over stripe code",
			in: api.TaxConfig{
				TaxCodeId: lo.ToPtr(inputID),
				Stripe:    &api.StripeTaxConfig{Code: stripeCode},
				Behavior:  lo.ToPtr(api.TaxBehaviorExclusive),
			},
			want: &productcatalog.TaxCodeConfig{TaxCodeID: inputID, Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)},
		},
		{
			name:      "stripe code is resolved to a tax code",
			in:        api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: stripeCode}},
			want:      &productcatalog.TaxCodeConfig{TaxCodeID: resolvedID},
			wantCalls: []productcatalog.ResolveOrBackfillTaxCodeInput{{Namespace: namespace, StripeCode: stripeCode}},
		},
		{
			name: "invalid stripe code is a validation error",
			in:   api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: "invalid"}},
			wantErr: func(t *testing.T, err error) {
				assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
			},
		},
		{
			name: "empty tax code ID is a validation error, not a fallback to the stripe code",
			in: api.TaxConfig{
				Stripe:    &api.StripeTaxConfig{Code: stripeCode},
				TaxCodeId: lo.ToPtr(""),
			},
			wantErr: func(t *testing.T, err error) {
				assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
			},
		},
		{
			name:        "resolver failure is not a validation error",
			in:          api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: stripeCode}},
			resolverErr: resolverErr,
			wantCalls:   []productcatalog.ResolveOrBackfillTaxCodeInput{{Namespace: namespace, StripeCode: stripeCode}},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, resolverErr)
				assert.False(t, models.IsGenericValidationError(err), "resolver failures must not become validation errors")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &stubBackfillResolver{id: resolvedID, err: tt.resolverErr}

			got, err := productcataloghttp.AsTaxCodeConfig(t.Context(), resolver, namespace, tt.in)

			assert.Equal(t, tt.wantCalls, resolver.calls)

			if tt.wantErr != nil {
				require.Error(t, err)
				tt.wantErr(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFromTaxCodeConfig(t *testing.T) {
	const id = "01JTAXCODE00000000000000000"

	taxCodes := map[string]*taxcode.TaxCode{
		id: {
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: id},
			AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
		},
	}

	tests := []struct {
		name     string
		cfg      productcatalog.TaxCodeConfig
		taxCodes map[string]*taxcode.TaxCode
		want     api.TaxConfig
	}{
		{
			name:     "stripe code from the referenced tax code",
			cfg:      productcatalog.TaxCodeConfig{TaxCodeID: id, Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)},
			taxCodes: taxCodes,
			want: api.TaxConfig{
				Behavior:  lo.ToPtr(api.TaxBehaviorInclusive),
				Stripe:    &api.StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeId: lo.ToPtr(id),
			},
		},
		{
			name: "tax code not in the map",
			cfg:  productcatalog.TaxCodeConfig{TaxCodeID: id},
			want: api.TaxConfig{TaxCodeId: lo.ToPtr(id)},
		},
		{
			name: "tax code without stripe mapping",
			cfg:  productcatalog.TaxCodeConfig{TaxCodeID: id},
			taxCodes: map[string]*taxcode.TaxCode{
				id: {NamespacedID: models.NamespacedID{Namespace: "ns", ID: id}},
			},
			want: api.TaxConfig{TaxCodeId: lo.ToPtr(id)},
		},
		{
			name:     "behavior only",
			cfg:      productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)},
			taxCodes: taxCodes,
			want:     api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, productcataloghttp.FromTaxCodeConfig(tt.cfg, tt.taxCodes))
		})
	}
}
