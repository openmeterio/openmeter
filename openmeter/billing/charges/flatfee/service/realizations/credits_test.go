package realizations

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type fiatOverageAllocationPolicyAdapter struct {
	flatfee.Adapter
	run flatfee.RealizationRunBase
}

func (a *fiatOverageAllocationPolicyAdapter) UpdateRealizationRun(
	_ context.Context,
	input flatfee.UpdateRealizationRunInput,
) (flatfee.RealizationRunBase, error) {
	if input.NoFiatTransactionRequired.IsPresent() {
		a.run.NoFiatTransactionRequired = input.NoFiatTransactionRequired.OrEmpty()
	}
	if input.FiatOverageCreditAllocationCompleted.IsPresent() {
		a.run.FiatOverageCreditAllocationCompleted = input.FiatOverageCreditAllocationCompleted.OrEmpty()
	}

	return a.run, nil
}

type uncalledFiatOverageHandler struct {
	flatfee.Handler
}

func TestAllocateFiatOverageCreditsSkipsHandlerWhenDisabled(t *testing.T) {
	// given:
	// - a prepared custom-currency overage with fiat credit allocation disabled
	// - a nil embedded handler which would panic if called
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-2 * time.Hour),
		To:   now.Add(-time.Hour),
	}
	customCurrency := currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	costBasisID := "cost-basis-1"
	lineID := "line-1"
	run := flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID:                   flatfee.RealizationRunID(models.NamespacedID{Namespace: "ns", ID: "run-1"}),
			ManagedModel:         models.ManagedModel{CreatedAt: now, UpdatedAt: now},
			LineID:               &lineID,
			Type:                 flatfee.RealizationRunTypeFinalRealization,
			InitialType:          flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod:        servicePeriod,
			AmountAfterProration: alpacadecimal.NewFromInt(10),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(10),
				Total:  alpacadecimal.NewFromInt(10),
			},
		},
		AccruedUsage: &invoicedusage.AccruedUsage{
			ServicePeriod: servicePeriod,
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(5),
				Total:  alpacadecimal.NewFromInt(5),
			},
		},
	}
	charge := flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ManagedModel:    models.ManagedModel{CreatedAt: now, UpdatedAt: now},
				ID:              "charge-1",
			},
			Intent: flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: "customer-1",
					Currency:   customCurrency,
					TaxConfig:  productcatalog.TaxCodeConfig{TaxCodeID: "tax-code-1"},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat fee",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.To,
					PaymentTerm:           productcatalog.InArrearsPaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}.AsOverridableIntent(),
			Status: flatfee.StatusActiveRealizationProcessing,
			State: flatfee.State{
				AmountAfterProration: alpacadecimal.NewFromInt(10),
				CostBasisID:          &costBasisID,
			},
		},
		Realizations: flatfee.Realizations{CurrentRun: &run},
	}

	adapter := &fiatOverageAllocationPolicyAdapter{run: run.RealizationRunBase}
	service := Service{
		adapter: adapter,
		handler: uncalledFiatOverageHandler{},
	}

	// when:
	// - charge orchestration completes the allocation phase without allowing fiat credits
	result, err := service.AllocateFiatOverageCredits(t.Context(), AllocateFiatOverageCreditsInput{
		Charge:           charge,
		Run:              run,
		AllowFiatCredits: false,
	})

	// then:
	// - no handler is invoked and the full fiat overage remains payable
	require.NoError(t, err)
	require.Empty(t, result.Run.FiatOverageCreditRealizations)
	require.True(t, result.Run.FiatOverageCreditAllocationCompleted)
	require.False(t, result.Run.NoFiatTransactionRequired)
}
