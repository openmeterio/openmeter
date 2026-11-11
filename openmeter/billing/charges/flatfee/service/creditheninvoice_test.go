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
	// - the cost basis is resolved before the realization run starts
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
	adapter := &attachBeforeActiveAdapter{errRunStarted: errRunStarted}
	resolver := &attachBeforeActiveCostBasisResolver{state: resolvedState}

	realizations, err := flatfeerealizations.New(flatfeerealizations.Config{
		Adapter:       adapter,
		Handler:       attachBeforeActiveHandler{},
		Lineage:       attachBeforeActiveLineage{},
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

	require.Len(t, adapter.runCharges, 1)
	require.Equal(t, &resolvedState, adapter.runCharges[0].State.ResolvedCostBasis)
}

type attachBeforeActiveAdapter struct {
	flatfee.Adapter

	errRunStarted error
	runCharges    []flatfee.ChargeBase
}

func (a *attachBeforeActiveAdapter) SetResolvedCostBasis(_ context.Context, input costbasis.SetResolvedCostBasisInput) (costbasis.CostBasis, error) {
	return costbasis.CostBasis{State: &input.State}, nil
}

func (a *attachBeforeActiveAdapter) CreateCurrentRun(_ context.Context, input flatfee.CreateCurrentRunInput) (flatfee.RealizationRunBase, error) {
	a.runCharges = append(a.runCharges, input.Charge)

	return flatfee.RealizationRunBase{}, a.errRunStarted
}

type attachBeforeActiveCostBasisResolver struct {
	costbasis.Resolver

	state              costbasis.State
	servicePeriodFroms []time.Time
}

func (r *attachBeforeActiveCostBasisResolver) ResolveDynamicState(_ context.Context, input costbasis.ResolveDynamicStateInput) (costbasis.State, error) {
	r.servicePeriodFroms = append(r.servicePeriodFroms, input.ServicePeriodFrom)

	return r.state, nil
}

type attachBeforeActiveHandler struct {
	flatfee.Handler
}

type attachBeforeActiveLineage struct {
	legacylineage.Service
}
