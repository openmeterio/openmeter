package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPopulateUsageBasedStandardLineFromRunProjectsDetailsAndCredits(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	line := newUsageBasedStandardLineForTest(period)
	line.DetailedLines = billing.DetailedLines{
		{
			DetailedLineBase: billing.DetailedLineBase{
				Base: stddetailedline.Base{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						ID:        "existing-detail-id",
						Namespace: line.Namespace,
						Name:      "existing",
					}),
					ChildUniqueReferenceID: "usage-a",
				},
			},
		},
	}

	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: line.Namespace,
				ID:        "run-id",
			},
			StoredAtLT:      period.To,
			ServicePeriodTo: period.To,
			MeteredQuantity: alpacadecimal.NewFromInt(20),
			Totals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(20),
				CreditsTotal: alpacadecimal.NewFromInt(7),
				Total:        alpacadecimal.NewFromInt(13),
			},
		},
		CreditsAllocated: creditrealization.Realizations{
			{
				NamespacedModel: models.NamespacedModel{
					Namespace: line.Namespace,
				},
				CreateInput: creditrealization.CreateInput{
					ID: "credit-realization-id",
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: "ledger-transaction-id",
					},
					ServicePeriod: period,
					Amount:        alpacadecimal.NewFromInt(7),
					Type:          creditrealization.TypeAllocation,
				},
			},
		},
		DetailedLines: mo.Some(usagebased.DetailedLines{
			newUsageBasedDetailedLineForTest("usage-a", period, alpacadecimal.NewFromInt(10)),
			newUsageBasedDetailedLineForTest("usage-b", period, alpacadecimal.NewFromInt(10)),
		}),
	}

	priorRun := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: line.Namespace,
				ID:        "prior-run-id",
			},
			Type:            usagebased.RealizationRunTypePartialInvoice,
			StoredAtLT:      period.From,
			ServicePeriodTo: period.From.Add(24 * time.Hour),
			MeteredQuantity: alpacadecimal.NewFromInt(5),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(5),
				Total:  alpacadecimal.NewFromInt(5),
			},
		},
	}

	err := populateUsageBasedStandardLineFromRun(line, run, usagebased.RealizationRuns{priorRun, run})
	require.NoError(t, err)

	require.Len(t, line.DetailedLines, 2)
	require.Equal(t, "existing-detail-id", line.DetailedLines[0].ID)
	require.Equal(t, line.InvoiceID, line.DetailedLines[0].InvoiceID)
	require.Equal(t, line.Namespace, line.DetailedLines[0].Namespace)
	require.Len(t, line.DetailedLines[0].CreditsApplied, 1)
	require.Equal(t, float64(7), line.DetailedLines[0].CreditsApplied[0].Amount.InexactFloat64())
	require.Equal(t, float64(3), line.DetailedLines[0].Totals.Total.InexactFloat64())
	require.Empty(t, line.DetailedLines[1].CreditsApplied)
	require.Equal(t, float64(10), line.DetailedLines[1].Totals.Total.InexactFloat64())
	require.Len(t, line.CreditsApplied, 1)
	require.Equal(t, float64(13), line.Totals.Total.InexactFloat64())
	require.Equal(t, float64(15), lo.FromPtr(line.UsageBased.Quantity).InexactFloat64())
	require.Equal(t, float64(15), lo.FromPtr(line.UsageBased.MeteredQuantity).InexactFloat64())
	require.Equal(t, float64(5), lo.FromPtr(line.UsageBased.PreLinePeriodQuantity).InexactFloat64())
	require.Equal(t, float64(5), lo.FromPtr(line.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())
}

func TestPopulateUsageBasedStandardLineFromRunAppliesUsageDiscount(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	line := newUsageBasedStandardLineForTest(period)
	line.RateCardDiscounts.Usage = &billing.UsageDiscount{
		UsageDiscount: productcatalog.UsageDiscount{
			Quantity: alpacadecimal.NewFromInt(10),
		},
		CorrelationID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}

	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: line.Namespace,
				ID:        "run-id",
			},
			StoredAtLT:      period.To,
			ServicePeriodTo: period.To,
			MeteredQuantity: alpacadecimal.NewFromInt(20),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(10),
				Total:  alpacadecimal.NewFromInt(10),
			},
		},
		DetailedLines: mo.Some(usagebased.DetailedLines{
			newUsageBasedDetailedLineForTest("usage-a", period, alpacadecimal.NewFromInt(10)),
		}),
	}

	priorRun := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: line.Namespace,
				ID:        "prior-run-id",
			},
			Type:            usagebased.RealizationRunTypePartialInvoice,
			StoredAtLT:      period.From,
			ServicePeriodTo: period.From.Add(24 * time.Hour),
			MeteredQuantity: alpacadecimal.NewFromInt(5),
		},
	}

	err := populateUsageBasedStandardLineFromRun(line, run, usagebased.RealizationRuns{priorRun, run})
	require.NoError(t, err)

	require.Equal(t, float64(10), lo.FromPtr(line.UsageBased.Quantity).InexactFloat64())
	require.Equal(t, float64(15), lo.FromPtr(line.UsageBased.MeteredQuantity).InexactFloat64())
	require.Equal(t, float64(0), lo.FromPtr(line.UsageBased.PreLinePeriodQuantity).InexactFloat64())
	require.Equal(t, float64(5), lo.FromPtr(line.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())

	require.Len(t, line.Discounts.Usage, 1)
	usageDiscount := line.Discounts.Usage[0]
	require.Equal(t, "rateCardDiscount/correlationID=01ARZ3NDEKTSV4RRFFQ69G5FAV", lo.FromPtr(usageDiscount.ChildUniqueReferenceID))
	require.Equal(t, float64(5), usageDiscount.Quantity.InexactFloat64())
	require.Equal(t, float64(5), lo.FromPtr(usageDiscount.PreLinePeriodQuantity).InexactFloat64())

	reason, err := usageDiscount.Reason.AsRatecardUsage()
	require.NoError(t, err)
	require.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", reason.CorrelationID)
	require.Equal(t, float64(10), reason.Quantity.InexactFloat64())
}

func TestPopulateUsageBasedStandardLineFromRunRequiresExpandedDetails(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	line := newUsageBasedStandardLineForTest(period)
	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: line.Namespace,
				ID:        "run-id",
			},
			StoredAtLT:      period.To,
			ServicePeriodTo: period.To,
			MeteredQuantity: alpacadecimal.NewFromInt(20),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(20),
				Total:  alpacadecimal.NewFromInt(20),
			},
		},
	}

	err := populateUsageBasedStandardLineFromRun(line, run, usagebased.RealizationRuns{run})
	require.ErrorContains(t, err, "detailed lines must be expanded")
}

func newUsageBasedStandardLineForTest(period timeutil.ClosedPeriod) *billing.StandardLine {
	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(1),
	})

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:        "line-id",
				Namespace: "namespace",
				Name:      "line",
			}),
			ManagedBy: billing.SystemManagedLine,
			Engine:    billing.LineEngineTypeChargeUsageBased,
			InvoiceID: "invoice-id",
			Currency:  currencyx.Code("USD"),
			Period:    period,
			InvoiceAt: period.To,
			ChargeID:  lo.ToPtr("charge-id"),
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      price,
			FeatureKey: "feature",
		},
	}
}

func newUsageBasedDetailedLineForTest(ref string, period timeutil.ClosedPeriod, amount alpacadecimal.Decimal) usagebased.DetailedLine {
	return usagebased.DetailedLine{
		PricerReferenceID: ref,
		Base: stddetailedline.Base{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "namespace",
				Name:      ref,
			}),
			Category:               stddetailedline.CategoryRegular,
			ChildUniqueReferenceID: ref,
			Index:                  lo.ToPtr(0),
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			ServicePeriod:          period,
			Currency:               currencyx.Code("USD"),
			PerUnitAmount:          amount,
			Quantity:               alpacadecimal.NewFromInt(1),
			Totals: totals.Totals{
				Amount: amount,
				Total:  amount,
			},
		},
	}
}
