package productcatalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// stubTaxCodeResolver serves BatchResolve from an in-memory map and records each call.
type stubTaxCodeResolver struct {
	taxCodes map[string]*taxcode.TaxCode
	err      error
	calls    [][]string
}

var _ productcatalog.TaxCodeResolver = (*stubTaxCodeResolver)(nil)

func (s *stubTaxCodeResolver) Resolve(context.Context, string, string) (*taxcode.TaxCode, error) {
	return nil, errors.New("not implemented")
}

func (s *stubTaxCodeResolver) BatchResolve(_ context.Context, _ string, ids ...string) (map[string]*taxcode.TaxCode, error) {
	s.calls = append(s.calls, ids)

	if s.err != nil {
		return nil, s.err
	}

	return lo.PickByKeys(s.taxCodes, ids), nil
}

func (s *stubTaxCodeResolver) WithNamespace(string) productcatalog.NamespacedTaxCodeResolver {
	return nil
}

func TestValidateRateCardTaxCodes(t *testing.T) {
	const (
		namespace = "ns"
		liveID    = "01JTAXCODE00000000000000000"
		deletedID = "01JTAXCODE00000000000000001"
		missingID = "01JTAXCODE00000000000000002"
	)

	taxCodes := map[string]*taxcode.TaxCode{
		liveID: {NamespacedID: models.NamespacedID{Namespace: namespace, ID: liveID}},
		deletedID: {
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deletedID},
			ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(clock.Now().Add(-1))},
		},
	}

	rateCard := func(key string, cfg *productcatalog.TaxCodeConfig) productcatalog.RateCard {
		return &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{Key: key, Name: key, TaxConfig: cfg},
		}
	}

	tests := []struct {
		name          string
		rateCards     productcatalog.RateCards
		resolverErr   error
		wantCalls     [][]string
		wantNotFound  bool
		wantErrNonVal bool
	}{
		{
			name:      "no tax configs skip the lookup",
			rateCards: productcatalog.RateCards{rateCard("a", nil)},
		},
		{
			name: "behavior only skips the lookup",
			rateCards: productcatalog.RateCards{
				rateCard("a", &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)}),
			},
		},
		{
			name:      "existing tax code",
			rateCards: productcatalog.RateCards{rateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: liveID})},
			wantCalls: [][]string{{liveID}},
		},
		{
			name:         "missing tax code",
			rateCards:    productcatalog.RateCards{rateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: missingID})},
			wantCalls:    [][]string{{missingID}},
			wantNotFound: true,
		},
		{
			name:         "deleted tax code",
			rateCards:    productcatalog.RateCards{rateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: deletedID})},
			wantCalls:    [][]string{{deletedID}},
			wantNotFound: true,
		},
		{
			name:          "resolver failure is not a validation error",
			rateCards:     productcatalog.RateCards{rateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: liveID})},
			resolverErr:   errors.New("db down"),
			wantCalls:     [][]string{{liveID}},
			wantErrNonVal: true,
		},
		{
			name: "shared tax code is looked up once",
			rateCards: productcatalog.RateCards{
				rateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: liveID}),
				rateCard("b", &productcatalog.TaxCodeConfig{TaxCodeID: liveID}),
			},
			wantCalls: [][]string{{liveID}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &stubTaxCodeResolver{taxCodes: taxCodes, err: tt.resolverErr}

			err := productcatalog.ValidateRateCardTaxCodes(t.Context(), resolver, namespace, tt.rateCards)

			assert.Equal(t, tt.wantCalls, resolver.calls)

			switch {
			case tt.wantNotFound:
				require.Error(t, err)
				assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
				assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)
			case tt.wantErrNonVal:
				require.Error(t, err)
				assert.False(t, models.IsGenericValidationError(err), "resolver failures must not become validation errors")
				assert.ErrorIs(t, err, tt.resolverErr)
			default:
				require.NoError(t, err)
			}
		})
	}
}
