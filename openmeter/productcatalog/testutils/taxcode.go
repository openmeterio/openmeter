package testutils

import (
	"context"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CountingTaxCodeResolver records BatchResolve calls and serves them from TaxCodes. Only
// BatchResolve is implemented; calling anything else panics on the nil embedded interface.
type CountingTaxCodeResolver struct {
	productcatalog.TransactionalTaxCodeResolver

	TaxCodes map[string]*taxcode.TaxCode
	Calls    [][]string
}

func (r *CountingTaxCodeResolver) BatchResolve(_ context.Context, _ string, ids ...string) (map[string]*taxcode.TaxCode, error) {
	r.Calls = append(r.Calls, ids)

	return lo.PickByKeys(r.TaxCodes, ids), nil
}

// NewTestFlatRateCard returns a billable flat fee rate card with the given Tax Code config.
func NewTestFlatRateCard(key string, tc *productcatalog.TaxCodeConfig) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:       key,
			Name:      key,
			TaxConfig: tc,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		BillingCadence: &MonthPeriod,
	}
}

// RequireTaxCodeNotFound asserts the error is the validation error for a Tax Code reference the
// namespace can't resolve (missing or from another namespace).
func RequireTaxCodeNotFound(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)
}

// RequireTaxCodeDeleted asserts the error is the validation error for a rate card referencing a
// soft-deleted Tax Code.
func RequireTaxCodeDeleted(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeDeleted)
}
