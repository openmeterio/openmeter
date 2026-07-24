package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
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
			Type:            usagebased.RealizationRunTypeFinalRealization,
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

	err := populateStandardLineFromRun(line, populateStandardLineFromRunInput{
		Charge: usagebased.Charge{
			Realizations: usagebased.RealizationRuns{priorRun, run},
		},
		Run: run,
	})
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
			Type:            usagebased.RealizationRunTypeFinalRealization,
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

	err := populateStandardLineFromRun(line, populateStandardLineFromRunInput{
		Charge: usagebased.Charge{
			Realizations: usagebased.RealizationRuns{priorRun, run},
		},
		Run: run,
	})
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
			Type:            usagebased.RealizationRunTypeFinalRealization,
			StoredAtLT:      period.To,
			ServicePeriodTo: period.To,
			MeteredQuantity: alpacadecimal.NewFromInt(20),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(20),
				Total:  alpacadecimal.NewFromInt(20),
			},
		},
	}

	err := populateStandardLineFromRun(line, populateStandardLineFromRunInput{
		Charge: usagebased.Charge{
			Realizations: usagebased.RealizationRuns{run},
		},
		Run: run,
	})
	require.ErrorContains(t, err, "detailed lines must be expanded")
}

func TestPopulateUsageBasedStandardLineFromCustomCurrencyRunCreatesFiatOverage(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(4).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromInt(2),
	})
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "namespace",
				},
				ID: "charge-id",
			},
			Intent: usagebased.Intent{
				Intent: meta.Intent{
					Currency: currencies.Currency{
						Currency: customCurrency,
					},
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}.AsOverridableIntent(),
			State: usagebased.State{
				ResolvedCostBasis: &costbasis.State{
					CostBasis:  alpacadecimal.NewFromInt(2),
					ResolvedAt: period.From,
				},
			},
		},
	}

	line := newUsageBasedStandardLineForTest(period)
	line.Annotations = models.Annotations{
		billing.AnnotationKeyReason: lo.ToPtr(billing.AnnotationValueReasonOveragePlaceholder),
	}
	line.UsageBased.UnitConfig = &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(100),
	}
	line.RateCardDiscounts = billing.Discounts{
		Usage: &billing.UsageDiscount{},
	}
	line.Discounts = billing.StandardLineDiscounts{
		Usage: billing.UsageLineDiscountsManaged{{}},
	}
	line.CreditsApplied = billing.CreditsApplied{{}}
	line.DetailedLines = billing.DetailedLines{
		{
			DetailedLineBase: billing.DetailedLineBase{
				Base: stddetailedline.Base{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						ID:        "existing-overage-id",
						Namespace: line.Namespace,
						Name:      "existing overage",
					}),
					ChildUniqueReferenceID: creditpurchase.CreditPurchaseChildUniqueReferenceID,
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
			Type:       usagebased.RealizationRunTypeFinalRealization,
			StoredAtLT: period.To,
			Totals: totals.Totals{
				Total: alpacadecimal.NewFromInt(3),
			},
		},
	}
	charge.Realizations = usagebased.RealizationRuns{run}

	err = populateStandardLineFromRun(line, populateStandardLineFromRunInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)

	require.Equal(t, productcatalog.FlatPriceType, line.UsageBased.Price.Type())
	require.Nil(t, line.UsageBased.UnitConfig)
	require.Equal(t, float64(1), lo.FromPtr(line.UsageBased.MeteredQuantity).InexactFloat64())
	require.Equal(t, float64(1), lo.FromPtr(line.UsageBased.Quantity).InexactFloat64())
	require.Equal(t, float64(0), lo.FromPtr(line.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())
	require.Equal(t, float64(0), lo.FromPtr(line.UsageBased.PreLinePeriodQuantity).InexactFloat64())
	require.True(t, line.RateCardDiscounts.IsEmpty())
	require.Empty(t, line.Discounts.Usage)
	require.Empty(t, line.CreditsApplied)
	require.Equal(t, period.To.Add(usagebased.InternalCollectionPeriod), *line.OverrideCollectionPeriodEnd)

	require.Len(t, line.DetailedLines, 1)
	require.Equal(t, "existing-overage-id", line.DetailedLines[0].ID)
	require.NoError(t, line.Validate())

	err = populateStandardLineFromRun(line, populateStandardLineFromRunInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)
	require.Len(t, line.DetailedLines, 1)
	require.Equal(t, "existing-overage-id", line.DetailedLines[0].ID)
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
			Currency:  currencyx.FiatCode("USD"),
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
			PerUnitAmount:          amount,
			Quantity:               alpacadecimal.NewFromInt(1),
			Totals: totals.Totals{
				Amount: amount,
				Total:  amount,
			},
		},
	}
}
