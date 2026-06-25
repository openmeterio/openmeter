package service

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesCreatesChargeBackedGatheringLines() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	featureKey := apiRequestsTotal.Feature.Key

	usageLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: ns,
				Name:      "manual usage",
			}),
			ManagedBy:     billing.ManuallyManagedLine,
			Engine:        billing.LineEngineTypeInvoice,
			Currency:      USD,
			ServicePeriod: servicePeriod,
			InvoiceAt:     servicePeriod.To,
			Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(2),
			})),
			FeatureKey: featureKey,
			RateCardDiscounts: billing.Discounts{
				Usage: &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{
						Quantity: alpacadecimal.NewFromInt(3),
					},
				},
			},
			ChildUniqueReferenceID: lo.ToPtr("manual-usage"),
		},
	}

	flatLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        servicePeriod,
		InvoiceAt:     servicePeriod.From,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "manual flat",
		Currency:      USD,
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
		RateCardDiscounts: billing.Discounts{
			Percentage: &billing.PercentageDiscount{
				PercentageDiscount: productcatalog.PercentageDiscount{
					Percentage: models.NewPercentage(10),
				},
			},
		},
	})
	flatLine.Engine = billing.LineEngineTypeInvoice
	flatLine.ChildUniqueReferenceID = lo.ToPtr("manual-flat")

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			usageLine,
			flatLine,
		},
	})
	s.NoError(err)
	s.Require().NotNil(result)
	s.NotEmpty(result.Invoice.ID)
	s.Require().Len(result.Lines, 2)

	s.Equal("manual usage", result.Lines[0].Name)
	s.Equal(billing.LineEngineTypeChargeUsageBased, result.Lines[0].Engine)
	s.Require().NotNil(result.Lines[0].ChargeID)

	s.Equal("manual flat", result.Lines[1].Name)
	s.Equal(billing.LineEngineTypeChargeFlatFee, result.Lines[1].Engine)
	s.Require().NotNil(result.Lines[1].ChargeID)

	usageCharge := s.mustGetChargeByID(meta.ChargeID{
		Namespace: ns,
		ID:        lo.FromPtr(result.Lines[0].ChargeID),
	})
	usageBasedCharge, err := usageCharge.AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, usageBasedCharge.Intent.SettlementMode)
	s.Equal(billing.ManuallyManagedLine, usageBasedCharge.Intent.ManagedBy)
	s.Equal("manual-usage", lo.FromPtr(usageBasedCharge.Intent.UniqueReferenceID))
	s.Equal(featureKey, usageBasedCharge.Intent.BaseLayer.FeatureKey)
	s.Require().NotNil(usageBasedCharge.Intent.BaseLayer.Discounts.Usage)
	s.Equal(float64(3), usageBasedCharge.Intent.BaseLayer.Discounts.Usage.Quantity.InexactFloat64())

	flatCharge := s.mustGetChargeByID(meta.ChargeID{
		Namespace: ns,
		ID:        lo.FromPtr(result.Lines[1].ChargeID),
	})
	flatFeeCharge, err := flatCharge.AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, flatFeeCharge.Intent.SettlementMode)
	s.Equal(billing.ManuallyManagedLine, flatFeeCharge.Intent.ManagedBy)
	s.Equal("manual-flat", lo.FromPtr(flatFeeCharge.Intent.UniqueReferenceID))
	s.Equal(productcatalog.InAdvancePaymentTerm, flatFeeCharge.Intent.BaseLayer.PaymentTerm)
	s.Equal(float64(10), flatFeeCharge.Intent.BaseLayer.AmountBeforeProration.InexactFloat64())
	s.Require().NotNil(flatFeeCharge.Intent.BaseLayer.PercentageDiscounts)
	s.Equal(float64(10), flatFeeCharge.Intent.BaseLayer.PercentageDiscounts.Percentage.InexactFloat64())
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRollsBackCreatedChargesOnFailure() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-rollback")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	zeroFlatLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        servicePeriod,
		InvoiceAt:     servicePeriod.From,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "manual zero flat",
		Currency:      USD,
		PerUnitAmount: alpacadecimal.Zero,
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	zeroFlatLine.Engine = billing.LineEngineTypeInvoice
	zeroFlatLine.ChildUniqueReferenceID = lo.ToPtr("manual-zero-flat")

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			zeroFlatLine,
		},
	})
	s.Nil(result)
	s.Require().Error(err)
	s.Contains(err.Error(), "no gathering lines were created")

	listed, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listed.Items)
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRejectsNonManualInput() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-policy")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	systemLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        servicePeriod,
		InvoiceAt:     servicePeriod.From,
		ManagedBy:     billing.SystemManagedLine,
		Name:          "system flat",
		Currency:      USD,
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})

	_, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			systemLine,
		},
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "managed by must be manual")

	subscriptionLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        servicePeriod,
		InvoiceAt:     servicePeriod.From,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "subscription flat",
		Currency:      USD,
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	subscriptionLine.Subscription = &billing.SubscriptionReference{
		SubscriptionID: "sub-1",
		PhaseID:        "phase-1",
		ItemID:         "item-1",
		BillingPeriod:  servicePeriod,
	}

	_, err = s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			subscriptionLine,
		},
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "subscription is not allowed")
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRollsBackPartialChargeLineResults() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-partial-rollback")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	usageLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: ns,
				Name:      "manual usage",
			}),
			ManagedBy:     billing.ManuallyManagedLine,
			Engine:        billing.LineEngineTypeInvoice,
			Currency:      USD,
			ServicePeriod: servicePeriod,
			InvoiceAt:     servicePeriod.To,
			Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(2),
			})),
			FeatureKey: apiRequestsTotal.Feature.Key,
		},
	}

	zeroFlatLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        servicePeriod,
		InvoiceAt:     servicePeriod.From,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "manual zero flat",
		Currency:      USD,
		PerUnitAmount: alpacadecimal.Zero,
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	zeroFlatLine.Engine = billing.LineEngineTypeInvoice

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			usageLine,
			zeroFlatLine,
		},
	})
	s.Nil(result)
	s.Require().Error(err)
	s.Contains(err.Error(), "gathering line was not created")

	listedCharges, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listedCharges.Items)

	listedInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{ns},
		Customers:  []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listedInvoices.Items)
}
