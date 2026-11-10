package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestAttachInvoiceLineResolvesDynamicCostBasisBeforeStartingRun(t *testing.T) {
	// given:
	// - a created flat fee charge with an unresolved dynamic cost basis
	// when:
	// - an invoice line is attached before the charge ever became active
	// then:
	// - the charge is persisted in early activation with the resolved cost basis
	// - the realization run is started from a charge that already carries it
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)
	intent := charge.Intent.GetBaseIntent()
	dynamicCostBasis := costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: fiatCurrency})
	intent.CostBasis = &dynamicCostBasis
	charge.Intent = intent.AsOverridableIntent()
	charge.Status = flatfee.StatusCreated
	charge.State.ResolvedCostBasis = nil

	resolvedState := costbasis.State{
		CostBasis:  alpacadecimal.NewFromFloat(0.25),
		ResolvedAt: servicePeriod.From,
	}
	errRunStarted := errors.New("run started")
	adapter := &earlyActivationAdapter{errRunStarted: errRunStarted}
	resolver := &earlyActivationCostBasisResolver{state: resolvedState}

	realizations, err := flatfeerealizations.New(flatfeerealizations.Config{
		Adapter:       adapter,
		Handler:       earlyActivationHandler{},
		Lineage:       earlyActivationLineage{},
		RatingService: &billabilityRatingService{},
	})
	require.NoError(t, err)

	machine, err := NewCreditThenInvoiceStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      adapter,
		Realizations: realizations,
		Service:      &service{costbasisResolver: resolver},
	})
	require.NoError(t, err)

	ctx, err := transaction.SetDriverOnContext(t.Context(), flatFeeBillabilityTransaction{})
	require.NoError(t, err)

	line := newFlatFeeStandardLineForTest(servicePeriod)
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(3),
		PaymentTerm: productcatalog.InArrearsPaymentTerm,
	})
	_, err = machine.FireAndAdvanceUntilInvoicePatchesOrStable(ctx, meta.TriggerAttachInvoiceLine, billing.StandardLineWithInvoiceHeader{
		Line:    line,
		Invoice: billing.StandardInvoice{StandardInvoiceBase: billing.StandardInvoiceBase{ID: line.InvoiceID}},
	})
	require.ErrorIs(t, err, errRunStarted)

	require.Equal(t, []time.Time{servicePeriod.From}, resolver.servicePeriodFroms)

	require.Len(t, adapter.persisted, 1)
	require.Equal(t, flatfee.StatusActiveEarlyActivation, adapter.persisted[0].Status)
	require.Equal(t, &resolvedState, adapter.persisted[0].State.ResolvedCostBasis)

	require.Len(t, adapter.runCharges, 1)
	require.Equal(t, &resolvedState, adapter.runCharges[0].State.ResolvedCostBasis)
}

type earlyActivationAdapter struct {
	flatfee.Adapter

	errRunStarted error
	persisted     []flatfee.ChargeBase
	runCharges    []flatfee.ChargeBase
}

func (a *earlyActivationAdapter) SetResolvedCostBasis(_ context.Context, input costbasis.SetResolvedCostBasisInput) (costbasis.CostBasis, error) {
	return costbasis.CostBasis{State: &input.State}, nil
}

func (a *earlyActivationAdapter) UpdateCharge(_ context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	a.persisted = append(a.persisted, charge)

	return charge, nil
}

func (a *earlyActivationAdapter) CreateCurrentRun(_ context.Context, input flatfee.CreateCurrentRunInput) (flatfee.RealizationRunBase, error) {
	a.runCharges = append(a.runCharges, input.Charge)

	return flatfee.RealizationRunBase{}, a.errRunStarted
}

type earlyActivationCostBasisResolver struct {
	costbasis.Resolver

	state              costbasis.State
	servicePeriodFroms []time.Time
}

func (r *earlyActivationCostBasisResolver) ResolveDynamicState(_ context.Context, input costbasis.ResolveDynamicStateInput) (costbasis.State, error) {
	r.servicePeriodFroms = append(r.servicePeriodFroms, input.ServicePeriodFrom)

	return r.state, nil
}

type earlyActivationHandler struct {
	flatfee.Handler
}

type earlyActivationLineage struct {
	legacylineage.Service
}
