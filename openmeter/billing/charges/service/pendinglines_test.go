package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesCreatesChargeBackedGatheringLines() {
	defer s.FlatFeeTestHandler.Reset()
	s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
		return nil, nil
	}

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
	clock.FreezeTime(servicePeriod.To)
	defer clock.UnFreeze()

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
			Currency:      currencyx.FiatCode(USD),
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
		InvoiceAt:     servicePeriod.To.Add(time.Hour),
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "manual flat",
		Currency:      currencyx.FiatCode(USD),
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
		Currency: currencyx.FiatCode(USD),
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
	usageBasedIntent := usageBasedCharge.Intent.GetBaseIntent()
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, usageBasedCharge.Intent.GetSettlementMode())
	s.Equal(billing.ManuallyManagedLine, usageBasedIntent.ManagedBy)
	s.Equal("manual-usage", lo.FromPtr(usageBasedCharge.Intent.GetUniqueReferenceID()))
	s.Equal(featureKey, usageBasedIntent.FeatureKey)
	s.Require().NotNil(usageBasedIntent.Discounts.Usage)
	s.Equal(float64(3), usageBasedIntent.Discounts.Usage.Quantity.InexactFloat64())
	s.NotEmpty(usageBasedIntent.Discounts.Usage.CorrelationID)
	s.Require().NotNil(result.Lines[0].RateCardDiscounts.Usage)
	s.Equal(usageBasedIntent.Discounts.Usage.CorrelationID, result.Lines[0].RateCardDiscounts.Usage.CorrelationID)

	flatCharge := s.mustGetChargeByID(meta.ChargeID{
		Namespace: ns,
		ID:        lo.FromPtr(result.Lines[1].ChargeID),
	})
	flatFeeCharge, err := flatCharge.AsFlatFeeCharge()
	s.NoError(err)
	flatFeeIntent := flatFeeCharge.Intent.GetBaseIntent()
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, flatFeeCharge.Intent.GetSettlementMode())
	s.Equal(billing.ManuallyManagedLine, flatFeeIntent.ManagedBy)
	s.Equal("manual-flat", lo.FromPtr(flatFeeCharge.Intent.GetUniqueReferenceID()))
	s.Equal(productcatalog.InAdvancePaymentTerm, flatFeeIntent.PaymentTerm)
	s.Equal(float64(10), flatFeeIntent.AmountBeforeProration.InexactFloat64())
	s.Require().NotNil(flatFeeIntent.PercentageDiscounts)
	s.Equal(float64(10), flatFeeIntent.PercentageDiscounts.Percentage.InexactFloat64())
	s.NotEmpty(flatFeeIntent.PercentageDiscounts.CorrelationID)
	s.Require().NotNil(result.Lines[1].RateCardDiscounts.Percentage)
	s.Equal(flatFeeIntent.PercentageDiscounts.CorrelationID, result.Lines[1].RateCardDiscounts.Percentage.CorrelationID)

	// Given the persisted gathering lines carry stale discount snapshots.
	staleUsageGatheringDiscounts := result.Lines[0].RateCardDiscounts.Clone()
	staleUsageGatheringDiscounts.Usage.CorrelationID = "stale-gathering-usage-discount"
	staleFlatFeeGatheringDiscounts := result.Lines[1].RateCardDiscounts.Clone()
	staleFlatFeeGatheringDiscounts.Percentage.CorrelationID = "stale-gathering-percentage-discount"
	persistedInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
		Invoice: result.Invoice.GetInvoiceID(),
		Expand:  billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
	})
	s.NoError(err)
	persistedUsageLine, found := lo.Find(persistedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.Engine == billing.LineEngineTypeChargeUsageBased
	})
	s.Require().True(found)
	persistedFlatFeeLine, found := lo.Find(persistedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.Engine == billing.LineEngineTypeChargeFlatFee
	})
	s.Require().True(found)
	encodedStaleUsageGatheringDiscounts, err := json.Marshal(staleUsageGatheringDiscounts)
	s.NoError(err)
	updateResult, err := s.TestDB.PGDriver.DB().ExecContext(ctx,
		`UPDATE billing_invoice_lines SET ratecard_discounts = $1 WHERE id = $2`,
		string(encodedStaleUsageGatheringDiscounts),
		persistedUsageLine.ID,
	)
	s.NoError(err)
	updatedRows, err := updateResult.RowsAffected()
	s.NoError(err)
	s.Equal(int64(1), updatedRows)
	encodedStaleFlatFeeGatheringDiscounts, err := json.Marshal(staleFlatFeeGatheringDiscounts)
	s.NoError(err)
	updateResult, err = s.TestDB.PGDriver.DB().ExecContext(ctx,
		`UPDATE billing_invoice_lines SET ratecard_discounts = $1 WHERE id = $2`,
		string(encodedStaleFlatFeeGatheringDiscounts),
		persistedFlatFeeLine.ID,
	)
	s.NoError(err)
	updatedRows, err = updateResult.RowsAffected()
	s.NoError(err)
	s.Equal(int64(1), updatedRows)

	// When billing collects the charge-backed usage line into a standard invoice.
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})
	// Then the standard line uses the effective charge discount identity while generating detailed lines.
	s.NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
	invoicedUsageLine := invoices[0].Lines.OrEmpty()[0]
	s.Require().NotNil(invoicedUsageLine.RateCardDiscounts.Usage)
	s.Equal(usageBasedIntent.Discounts.Usage.CorrelationID, invoicedUsageLine.RateCardDiscounts.Usage.CorrelationID)

	// When billing reaches the flat-fee line's invoice time and collects it.
	clock.FreezeTime(flatLine.InvoiceAt)
	defer clock.UnFreeze()
	invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(flatLine.InvoiceAt),
	})
	// Then the standard line uses the effective charge discount identity instead of the stale snapshot.
	s.NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
	invoicedFlatFeeLine := invoices[0].Lines.OrEmpty()[0]
	s.Require().NotNil(invoicedFlatFeeLine.RateCardDiscounts.Percentage)
	s.Equal(flatFeeIntent.PercentageDiscounts.CorrelationID, invoicedFlatFeeLine.RateCardDiscounts.Percentage.CorrelationID)
}

func (s *InvoicableChargesTestSuite) TestMissingFeatureCreatesInvalidStandardInvoiceForChargeBackedLine() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("charges-service-missing-feature-standard-invoice")
	s.ProvisionDefaultTaxCodes(ctx, namespace)

	customInvoicing := s.SetupCustomInvoicing(namespace)
	customerEntity := s.CreateTestCustomer(namespace, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.To)
	defer clock.UnFreeze()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, namespace)
	featureKey := apiRequestsTotal.Feature.Key

	// given: a charge-backed gathering line references a feature that existed when the charge was created
	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: customerEntity.GetID(),
		Currency: currencyx.FiatCode(USD),
		Lines: []billing.GatheringLine{{
			GatheringLineBase: billing.GatheringLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: namespace,
					Name:      "charge-backed usage with missing feature",
				}),
				ManagedBy:     billing.ManuallyManagedLine,
				Engine:        billing.LineEngineTypeInvoice,
				Currency:      currencyx.FiatCode(USD),
				ServicePeriod: servicePeriod,
				InvoiceAt:     servicePeriod.To,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				})),
				FeatureKey: featureKey,
			},
		}},
	})
	s.Require().NoError(err)
	s.Require().Len(result.Lines, 1)
	s.Equal(billing.LineEngineTypeChargeUsageBased, result.Lines[0].Engine)

	missingFeatureKey := "missing-feature"
	updateResult, err := s.TestDB.PGDriver.DB().ExecContext(ctx,
		`UPDATE billing_invoice_usage_based_line_configs SET feature_key = $1 WHERE id = $2`,
		missingFeatureKey,
		result.Lines[0].UBPConfigID,
	)
	s.Require().NoError(err)
	updatedRows, err := updateResult.RowsAffected()
	s.Require().NoError(err)
	s.Equal(int64(1), updatedRows)

	// when: billing collects the gathering line after its persisted feature reference becomes dangling
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})

	// then: the charge engine returns its usable line with a validation issue instead of aborting collection
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	invoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftInvalidCreated, invoice.Status)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(result.Lines[0].ID, invoice.Lines.OrEmpty()[0].ID)
	s.Require().Len(invoice.ValidationIssues, 1)
	s.Equal(billing.ErrInvoiceLineFeatureNotFound.Code, invoice.ValidationIssues[0].Code)
	s.Equal(billing.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
	s.Equal(billing.LineEngineValidationComponent(billing.LineEngineTypeChargeUsageBased), invoice.ValidationIssues[0].Component)
	s.Equal("/lines/"+result.Lines[0].ID, invoice.ValidationIssues[0].Path)
}

func (s *InvoicableChargesTestSuite) TestBillingCreatePendingInvoiceLinesResolvesRequiredFeatureMeters() {
	// given:
	// - usage pending lines whose feature dependencies are missing or meterless
	// when:
	// - each pending line is created through the billing service
	// then:
	// - validation rejects each request without creating a gathering invoice
	testCases := []struct {
		name          string
		createFeature bool
		expectedCode  string
		expectedError string
	}{
		{
			name:          "missing feature",
			expectedCode:  billing.ErrInvoiceLineFeatureNotFound.Code,
			expectedError: "not found",
		},
		{
			name:          "feature without meter",
			createFeature: true,
			expectedCode:  billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			expectedError: "usage based invoice line: feature has no meters",
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.name, func() {
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("billing-create-pending-line-feature-meter")
			cust := s.CreateTestCustomer(ns, "test-subject")
			customInvoicing := s.SetupCustomInvoicing(ns)
			_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID())

			featureKey := "pending-line-feature"
			if testCase.createFeature {
				_, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
					Namespace: ns,
					Name:      featureKey,
					Key:       featureKey,
				})
				s.Require().NoError(err)
			}

			servicePeriod := timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			}
			line := billing.GatheringLine{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: ns,
						Name:      "manual usage",
					}),
					ManagedBy:     billing.ManuallyManagedLine,
					Currency:      currencyx.FiatCode(USD),
					ServicePeriod: servicePeriod,
					InvoiceAt:     servicePeriod.To,
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(2),
					})),
					FeatureKey: featureKey,
				},
			}

			result, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: cust.GetID(),
				Currency: currencyx.FiatCode(USD),
				Lines:    []billing.GatheringLine{line},
			})

			s.Nil(result)
			s.Require().Error(err)
			var validationErr billing.ValidationError
			s.True(errors.As(err, &validationErr), "expected billing validation error, got %T: %v", err, err)
			issue := requireFeatureMeterValidationIssue(s.T(), err, testCase.expectedCode)
			s.Empty(issue.Path)
			s.ErrorContains(err, testCase.expectedError)

			listed, listErr := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
				Namespace: ns,
				Customers: []string{cust.ID},
			})
			s.Require().NoError(listErr)
			s.Empty(listed.Items)
		})
	}
}

func (s *InvoicableChargesTestSuite) TestChargeCreatePendingInvoiceLinesRequiresFeatureMeter() {
	// given:
	// - a charge-backed usage pending line references a feature without a meter
	// when:
	// - the pending line is created through the charge service
	// then:
	// - validation rejects the request without creating a charge or gathering invoice
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charge-create-pending-line-feature-meter")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	featureKey := "pending-line-feature"
	_, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: ns,
		Name:      featureKey,
		Key:       featureKey,
	})
	s.Require().NoError(err)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: ns,
				Name:      "manual usage",
			}),
			ManagedBy:     billing.ManuallyManagedLine,
			Engine:        billing.LineEngineTypeInvoice,
			Currency:      currencyx.FiatCode(USD),
			ServicePeriod: servicePeriod,
			InvoiceAt:     servicePeriod.To,
			Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(2),
			})),
			FeatureKey: featureKey,
		},
	}

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
		Lines:    []billing.GatheringLine{line},
	})

	s.Nil(result)
	s.Require().Error(err)
	issue := requireFeatureMeterValidationIssue(s.T(), err, billing.ErrInvoiceLineFeatureHasNoMeters.Code)
	s.Empty(issue.Path)
	s.ErrorContains(err, "usage based invoice line: feature has no meters")

	listedCharges, listChargesErr := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.Require().NoError(listChargesErr)
	s.Empty(listedCharges.Items)

	listedInvoices, listInvoicesErr := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespace: ns,
		Customers: []string{cust.ID},
	})
	s.Require().NoError(listInvoicesErr)
	s.Empty(listedInvoices.Items)
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
		Currency:      currencyx.FiatCode(USD),
		PerUnitAmount: alpacadecimal.Zero,
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	zeroFlatLine.Engine = billing.LineEngineTypeInvoice
	zeroFlatLine.ChildUniqueReferenceID = lo.ToPtr("manual-zero-flat")

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
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
		Currency:      currencyx.FiatCode(USD),
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})

	_, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
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
		Currency:      currencyx.FiatCode(USD),
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
		Currency: currencyx.FiatCode(USD),
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
			Currency:      currencyx.FiatCode(USD),
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
		Currency:      currencyx.FiatCode(USD),
		PerUnitAmount: alpacadecimal.Zero,
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	zeroFlatLine.Engine = billing.LineEngineTypeInvoice

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
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
		Namespace: ns,
		Customers: []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listedInvoices.Items)
}
