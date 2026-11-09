package httpdriver

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// recordingBackfillResolver serves ResolveOrBackfillTaxCode from taxCodes and records the codes.
type recordingBackfillResolver struct {
	taxCodes map[string]*taxcode.TaxCode
	err      error
	calls    []string
}

func (r *recordingBackfillResolver) ResolveOrBackfillTaxCode(_ context.Context, input productcatalog.ResolveOrBackfillTaxCodeInput) (*taxcode.TaxCode, error) {
	r.calls = append(r.calls, input.StripeCode)

	if r.err != nil {
		return nil, r.err
	}

	return r.taxCodes[input.StripeCode], nil
}

func TestResolveProfileDefaultTaxConfig(t *testing.T) {
	const namespace = "ns"

	storedTaxCode := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: "01JTAXCODE0000000000000000A"},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
	}
	// Another live Tax Code with the same Stripe mapping; resolving by mapping would return it.
	otherTaxCode := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: "01JTAXCODE0000000000000000B"},
		AppMappings:  storedTaxCode.AppMappings,
	}

	// given: a stored default rendered the way responses show it
	stored := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		TaxCodeID: lo.ToPtr(storedTaxCode.ID),
	}

	t.Run("create rejects a tax code without resolving it", func(t *testing.T) {
		// when: a stripe code is sent with nothing stored
		resolver := &recordingBackfillResolver{}
		_, err := ResolveProfileDefaultTaxConfig(t.Context(), resolver, namespace, &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}, nil)

		// then: the gate rejects it with a validation error and nothing is resolved
		var validationErr billing.ValidationError
		require.ErrorAs(t, err, &validationErr)
		require.Empty(t, resolver.calls)
	})

	t.Run("a stripe-only echo keeps the stored tax code ID", func(t *testing.T) {
		// when: a legacy client echoes the stored stripe code with a new behavior
		resolver := &recordingBackfillResolver{taxCodes: map[string]*taxcode.TaxCode{"txcd_10000000": otherTaxCode}}
		cfg, err := ResolveProfileDefaultTaxConfig(t.Context(), resolver, namespace, &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}, stored)
		require.NoError(t, err)

		// then: the stored ID is reused instead of looking the code up by mapping
		require.Equal(t, &productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: storedTaxCode.ID,
		}, cfg)
		require.Empty(t, resolver.calls)
	})

	t.Run("a stripe-only echo with no stored ID falls back to the resolver", func(t *testing.T) {
		// when: the stored default has a stripe code but no tax code ID
		resolver := &recordingBackfillResolver{taxCodes: map[string]*taxcode.TaxCode{"txcd_10000000": storedTaxCode}}
		cfg, err := ResolveProfileDefaultTaxConfig(t.Context(), resolver, namespace, &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}, &productcatalog.TaxConfig{Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"}})
		require.NoError(t, err)

		// then: the code is resolved by mapping
		require.Equal(t, &productcatalog.TaxCodeConfig{TaxCodeID: storedTaxCode.ID}, cfg)
		require.Equal(t, []string{"txcd_10000000"}, resolver.calls)
	})

	t.Run("resolver failures are passed through, not reported as validation errors", func(t *testing.T) {
		// when: resolving the fallback stripe code fails
		resolver := &recordingBackfillResolver{err: errors.New("db down")}
		_, err := ResolveProfileDefaultTaxConfig(t.Context(), resolver, namespace, &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		}, &productcatalog.TaxConfig{Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"}})

		// then: it's a TaxCodeResolutionError, not a billing.ValidationError
		var resolveErr *productcataloghttp.TaxCodeResolutionError
		require.ErrorAs(t, err, &resolveErr)

		var validationErr billing.ValidationError
		require.False(t, errors.As(err, &validationErr))
	})

	t.Run("an echoed tax code ID wins without resolving", func(t *testing.T) {
		// when: both the stored ID and stripe code are echoed
		resolver := &recordingBackfillResolver{}
		cfg, err := ResolveProfileDefaultTaxConfig(t.Context(), resolver, namespace, stored, stored)
		require.NoError(t, err)

		// then: the stored reference is kept
		require.Equal(t, &productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: storedTaxCode.ID,
		}, cfg)
		require.Empty(t, resolver.calls)
	})

	t.Run("dropping the stripe code while echoing the ID removes the pair", func(t *testing.T) {
		// when: the stripe code is dropped but the ID echoed
		resolver := &recordingBackfillResolver{}
		cfg, err := ResolveProfileDefaultTaxConfig(t.Context(), resolver, namespace, &productcatalog.TaxConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr(storedTaxCode.ID),
		}, stored)
		require.NoError(t, err)

		// then: only the behavior is left
		require.Equal(t, &productcatalog.TaxCodeConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		}, cfg)
	})

	t.Run("a malformed stripe code is a validation error", func(t *testing.T) {
		// when: the stripe code doesn't look like a Stripe tax code
		_, err := ResolveProfileDefaultTaxConfig(t.Context(), &recordingBackfillResolver{}, namespace, &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{Code: "not-a-code"},
		}, stored)

		// then: it's rejected as invalid input
		var validationErr billing.ValidationError
		require.ErrorAs(t, err, &validationErr)
		require.ErrorContains(t, err, "invalid default tax config")
	})
}
