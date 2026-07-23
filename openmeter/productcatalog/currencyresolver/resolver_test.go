package currencyresolver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/currencyresolver"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestResolveCurrency(t *testing.T) {
	// given:
	// - a code-only authoring currency
	// when:
	// - the currency is resolved before persistence
	// then:
	// - the managed identity returned by the shared resolver is used
	customCode := currencyx.Code("CREDITS")
	customID := "01J00000000000000000000000"
	resolver := &recordingCurrencyResolver{currencies: map[currencies.CurrencyRef]*currencies.Currency{
		{Code: customCode}: mustCurrency(t, customID, customCode),
	}}

	resolved, err := currencyresolver.ResolveCurrency(t.Context(), resolver, "namespace", customCode)
	require.NoError(t, err)
	managed, ok := resolved.(currencyx.ManagedCurrency)
	require.True(t, ok)
	require.Equal(t, customID, managed.GetID())
	require.Equal(t, 1, resolver.resolveCalls)
	require.Zero(t, resolver.batchCalls)
}

func TestResolveCurrencyRetainsManagedIdentity(t *testing.T) {
	customCode := currencyx.Code("CREDITS")
	oldID := "01J00000000000000000000000"
	newID := "01J00000000000000000000001"
	existing := mustCurrency(t, oldID, customCode)
	resolver := &recordingCurrencyResolver{currencies: map[currencies.CurrencyRef]*currencies.Currency{
		{Code: customCode}: mustCurrency(t, newID, customCode),
	}}

	resolved, err := currencyresolver.ResolveCurrency(t.Context(), resolver, "namespace", existing)
	require.NoError(t, err)
	managed, ok := resolved.(currencyx.ManagedCurrency)
	require.True(t, ok)
	require.Equal(t, oldID, managed.GetID())
	require.Zero(t, resolver.resolveCalls)
	require.Zero(t, resolver.batchCalls)
}

func TestResolveCurrenciesForRateCardsRetainsManagedIdentityAndBatchesAuthoringCodes(t *testing.T) {
	customCode := currencyx.Code("CREDITS")
	oldID := "01J00000000000000000000000"
	newID := "01J00000000000000000000001"
	oldIdentity := mustCurrency(t, oldID, customCode)
	resolver := &recordingCurrencyResolver{currencies: map[currencies.CurrencyRef]*currencies.Currency{
		{Code: customCode}: mustCurrency(t, newID, customCode),
	}}
	rateCards := productcatalog.RateCards{
		newRateCard("persisted", oldIdentity),
		newRateCard("new", customCode),
	}

	err := currencyresolver.ResolveCurrenciesForRateCards(t.Context(), resolver, "namespace", &rateCards)
	require.NoError(t, err)
	persisted := rateCards[0].AsMeta().Currency.(currencyx.ManagedCurrency)
	newCurrency := rateCards[1].AsMeta().Currency.(currencyx.ManagedCurrency)
	require.Equal(t, oldID, persisted.GetID())
	require.Equal(t, newID, newCurrency.GetID())
	require.Equal(t, 1, resolver.batchCalls)
	require.Equal(t, []currencies.CurrencyRef{{Code: customCode}}, resolver.lastBatch)
}

func TestResolveCurrenciesForRateCardsReportsMissingCurrency(t *testing.T) {
	rateCards := productcatalog.RateCards{newRateCard("missing", currencyx.Code("CREDITS"))}
	resolver := &recordingCurrencyResolver{}

	err := currencyresolver.ResolveCurrenciesForRateCards(t.Context(), resolver, "namespace", &rateCards)
	require.ErrorIs(t, err, productcatalog.ErrCurrencyNotFound)
}

func newRateCard(key string, identity currencyx.CurrencyIdentity) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
		Key:      key,
		Name:     key,
		Currency: identity,
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
	}}
}

func mustCurrency(t *testing.T, id string, code currencyx.Code) *currencies.Currency {
	t.Helper()

	currencyType := currencyx.CurrencyTypeCustom
	if code.IsFiat() {
		currencyType = currencyx.CurrencyTypeFiat
	}

	resolved, err := currencyx.NewCurrencyBuilder(currencyType).
		WithCode(code).
		WithName(code.String()).
		Build()
	require.NoError(t, err)

	return &currencies.Currency{
		NamespacedID: models.NamespacedID{ID: id},
		Currency:     resolved,
	}
}

type recordingCurrencyResolver struct {
	currencies   map[currencies.CurrencyRef]*currencies.Currency
	resolveCalls int
	batchCalls   int
	lastBatch    []currencies.CurrencyRef
}

func (r *recordingCurrencyResolver) ResolveCurrency(_ context.Context, _ string, ref currencies.CurrencyRef) (*currencies.Currency, error) {
	r.resolveCalls++

	resolved := r.currencies[ref]
	if resolved == nil {
		return nil, models.NewGenericNotFoundError(fmt.Errorf("currency %v", ref))
	}

	return resolved, nil
}

func (r *recordingCurrencyResolver) BatchResolveCurrencies(_ context.Context, _ string, refs ...currencies.CurrencyRef) (map[currencies.CurrencyRef]*currencies.Currency, error) {
	r.batchCalls++
	r.lastBatch = append([]currencies.CurrencyRef(nil), refs...)

	resolved := make(map[currencies.CurrencyRef]*currencies.Currency, len(refs))
	for _, ref := range refs {
		resolved[ref] = r.currencies[ref]
	}

	return resolved, nil
}

func (r *recordingCurrencyResolver) WithNamespace(namespace string) currencies.NamespacedCurrencyResolver {
	return &recordingNamespacedCurrencyResolver{resolver: r, namespace: namespace}
}

type recordingNamespacedCurrencyResolver struct {
	resolver  *recordingCurrencyResolver
	namespace string
}

func (r *recordingNamespacedCurrencyResolver) ResolveCurrency(ctx context.Context, ref currencies.CurrencyRef) (*currencies.Currency, error) {
	return r.resolver.ResolveCurrency(ctx, r.namespace, ref)
}

func (r *recordingNamespacedCurrencyResolver) BatchResolveCurrencies(ctx context.Context, refs ...currencies.CurrencyRef) (map[currencies.CurrencyRef]*currencies.Currency, error) {
	return r.resolver.BatchResolveCurrencies(ctx, r.namespace, refs...)
}

func (r *recordingNamespacedCurrencyResolver) Namespace() string {
	return r.namespace
}

var _ currencies.CurrencyResolver = (*recordingCurrencyResolver)(nil)
