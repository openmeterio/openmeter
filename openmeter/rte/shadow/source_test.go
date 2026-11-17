package shadow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	entitlementservice "github.com/openmeterio/openmeter/openmeter/entitlement/service"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/shadow"
	"github.com/openmeterio/openmeter/pkg/models"
)

type sourceObservations struct {
	creditReads  []customerbalance.GetBalanceServiceInput
	meteredReads []models.NamespacedID
	calculations []models.NamespacedID
}

func (r *sourceObservations) ObserveCreditRead(_ context.Context, input customerbalance.GetBalanceServiceInput, _ time.Duration) {
	r.creditReads = append(r.creditReads, input)
}

func (r *sourceObservations) ObserveMeteredRead(_ context.Context, id models.NamespacedID, _ time.Duration) {
	r.meteredReads = append(r.meteredReads, id)
}

func (r *sourceObservations) ObserveMeteredCalculation(_ context.Context, id models.NamespacedID, _ time.Duration) {
	r.calculations = append(r.calculations, id)
}

type creditSource struct {
	customerbalance.NoopService
	currencies []currencies.CurrencyReference
	failure    error
}

func (s *creditSource) GetBalanceCurrencies(context.Context, customerbalance.GetBalanceCurrenciesInput) ([]currencies.CurrencyReference, error) {
	return s.currencies, nil
}

func (s *creditSource) GetBalance(ctx context.Context, input customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
	if s.failure != nil {
		return nil, s.failure
	}

	return s.NoopService.GetBalance(ctx, input)
}

func TestCurrentCreditReads(t *testing.T) {
	// Given one current portfolio with two currency identities.
	source := &creditSource{currencies: []currencies.CurrencyReference{
		{Code: "USD"},
		{Code: "CUS", CustomCurrencyID: lo.ToPtr("currency-id")},
	}}
	var observed sourceObservations
	facade, err := customerbalance.NewFacade(shadow.ObserveCreditReads(source, &observed))
	require.NoError(t, err)
	input := customerbalance.GetBalancesInput{
		CustomerID:    customer.CustomerID{Namespace: "ns", ID: "customer"},
		FeatureFilter: customerbalance.NewFeatureFilter([]string{"storage"}),
	}

	// When the facade calculates each currency, the observer sees both resolved keys.
	values, err := facade.GetBalances(t.Context(), input)
	require.NoError(t, err)
	require.Len(t, values, 2)
	require.Len(t, observed.creditReads, 2)
	for i, read := range observed.creditReads {
		require.Equal(t, source.currencies[i], read.Currency)
		require.Equal(t, input.CustomerID, read.CustomerID)
		require.Equal(t, input.FeatureFilter, read.FeatureFilter)
	}

	// An explicit cutoff and a failed calculation must not generate current demand.
	input.AsOf = lo.ToPtr(time.Now())
	_, err = facade.GetBalances(t.Context(), input)
	require.NoError(t, err)

	input.AsOf = nil
	source.failure = errors.New("source unavailable")
	_, err = facade.GetBalances(t.Context(), input)
	require.ErrorIs(t, err, source.failure)
	require.Len(t, observed.creditReads, 2)
}

type entitlementByFeature struct {
	entitlement.EntitlementRepo
	ent *entitlement.Entitlement
}

func (r entitlementByFeature) GetActiveEntitlementOfCustomerAt(context.Context, string, string, string, time.Time) (*entitlement.Entitlement, error) {
	return r.ent, nil
}

type meteredSource struct {
	meteredentitlement.Connector
	value *meteredentitlement.MeteredEntitlementValue
	err   error
}

func (s *meteredSource) GetValue(context.Context, *entitlement.Entitlement, time.Time) (entitlement.EntitlementValue, error) {
	return s.value, s.err
}

func TestCurrentMeteredOperations(t *testing.T) {
	// Given a product lookup by feature that resolves to a metered entitlement.
	id := models.NamespacedID{Namespace: "ns", ID: "entitlement"}
	ent := &entitlement.Entitlement{GenericProperties: entitlement.GenericProperties{
		NamespacedModel: models.NamespacedModel{Namespace: id.Namespace},
		ID:              id.ID, EntitlementType: entitlement.EntitlementTypeMetered,
	}}
	source := &meteredSource{value: &meteredentitlement.MeteredEntitlementValue{Balance: 23}}
	var observed sourceObservations
	svc := entitlementservice.NewEntitlementService(entitlementservice.ServiceConfig{
		EntitlementRepo:             entitlementByFeature{ent: ent},
		BooleanEntitlementConnector: booleanentitlement.NewBooleanEntitlementConnector(),
		MeteredEntitlementConnector: shadow.ObserveMeteredValues(source, &observed),
	})

	// When read, calculated by the worker, or explicitly evaluated at the same time.
	at := time.Now()
	read := entitlement.CurrentValueOperationRead.WithContext(t.Context())
	calculate := entitlement.CurrentValueOperationCalculate.WithContext(t.Context())
	for _, ctx := range []context.Context{read, calculate, t.Context()} {
		value, err := svc.GetEntitlementValue(ctx, id.Namespace, "customer", "storage", at)
		require.NoError(t, err)
		require.Same(t, source.value, value)
	}

	source.err = errors.New("source unavailable")
	_, err := svc.GetEntitlementValue(read, id.Namespace, "customer", "storage", at)
	require.ErrorIs(t, err, source.err)

	ent.EntitlementType = entitlement.EntitlementTypeBoolean
	_, err = svc.GetEntitlementValue(read, id.Namespace, "customer", "storage", at)
	require.NoError(t, err)

	// Then only the two successful current metered operations record a canonical ID.
	require.Equal(t, []models.NamespacedID{id}, observed.meteredReads)
	require.Equal(t, []models.NamespacedID{id}, observed.calculations)
}
