package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/billinginvoices"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type invoiceNowErrorBillingService struct {
	billing.Service
	err   error
	calls int
}

func (s *invoiceNowErrorBillingService) InvoicePendingLines(context.Context, billing.InvoicePendingLinesInput, ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
	s.calls++
	return nil, s.err
}

func TestInvoiceNowOnCreateDefersFullyGatedCollection(t *testing.T) {
	unexpectedErr := errors.New("unexpected collection failure")
	tests := []struct {
		name      string
		err       error
		wantErr   error
		wantCalls int
	}{
		{name: "fully gated customers are deferred", err: fmt.Errorf("wrapped: %w", billing.ValidationError{Err: billing.ErrInvoiceCreateAllLinesGated}), wantCalls: 2},
		{name: "no eligible lines remains an error", err: billing.ValidationError{Err: billing.ErrInvoiceCreateNoLines}, wantErr: billing.ErrInvoiceCreateNoLines, wantCalls: 1},
		{name: "operational failure", err: unexpectedErr, wantErr: unexpectedErr, wantCalls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			billingService := &invoiceNowErrorBillingService{err: test.err}
			svc := &service{billingService: billingService}

			err := svc.invokeInvoiceNowOnCreate(t.Context(), []invoicePendingLinesInput{
				{CustomerID: customer.CustomerID{Namespace: "ns", ID: "customer-1"}, LineID: "line-1"},
				{CustomerID: customer.CustomerID{Namespace: "ns", ID: "customer-2"}, LineID: "line-2"},
			})
			require.ErrorIs(t, err, test.wantErr)
			require.Equal(t, test.wantCalls, billingService.calls)
		})
	}
}

func (s *InvoicableChargesTestSuite) TestInvoicePendingLinesAllGatedReturnsErrorAndPersistsChargeIssues() {
	tests := []struct {
		name                 string
		enclosingTransaction bool
	}{
		{name: "standalone transaction"},
		{name: "caller handles deferral in enclosing transaction", enclosingTransaction: true},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			// given: a due charge's required meter becomes unavailable.
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("all-gated-charge-validation")
			s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
			customInvoicing := s.SetupCustomInvoicing(ns)
			s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(), billingtest.WithManualApproval())
			cust := s.CreateTestCustomer(ns, "test-subject")
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(period.To)
			defer clock.UnFreeze()

			featureKey := "temporarily-unavailable-meter"
			featureMeters := s.createFeatureMeters(ctx, ns, featureKey)
			meters := []meter.Meter{*featureMeters.ByKey[featureKey].Meter}
			pending, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
				Customer: cust.GetID(),
				Currency: currencyx.FiatCode(USD),
				Lines: billing.CreatePendingInvoiceLines{{
					GatheringLine: billing.GatheringLine{
						GatheringLineBase: billing.GatheringLineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Namespace: ns,
								Name:      "blocked usage",
							}),
							ManagedBy:     billing.ManuallyManagedLine,
							Currency:      currencyx.FiatCode(USD),
							ServicePeriod: period,
							InvoiceAt:     period.To,
							FeatureKey:    featureKey,
							Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount: alpacadecimal.NewFromInt(2),
							}),
						},
					},
				}},
			})
			s.Require().NoError(err)
			s.Require().Len(pending.Lines, 1)
			s.Require().NotNil(pending.Lines[0].ChargeID)
			s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, nil))
			defer func() {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, meters))
			}()
			chargeID := meta.ChargeID{Namespace: ns, ID: *pending.Lines[0].ChargeID}
			input := billing.InvoicePendingLinesInput{Customer: cust.GetID(), AsOf: &period.To}

			// when: collection is fully gated, including when the caller owns the transaction.
			var collectionErr error
			if test.enclosingTransaction {
				_, err = transaction.Run(ctx, s.BillingAdapter, func(ctx context.Context) (struct{}, error) {
					_, collectionErr = s.BillingService.InvoicePendingLines(ctx, input, billing.WithBypassCollectionAlignment())
					if errors.Is(collectionErr, billing.ErrInvoiceCreateAllLinesGated) {
						return struct{}{}, nil
					}

					return struct{}{}, collectionErr
				})
				s.Require().NoError(err)
			} else {
				_, collectionErr = s.BillingService.InvoicePendingLines(ctx, input, billing.WithBypassCollectionAlignment())
			}

			// then: collection returns an error, the charge stores its validation issues, and the line remains pending.
			s.Require().ErrorIs(collectionErr, billing.ErrInvoiceCreateAllLinesGated)
			s.ErrorAs(collectionErr, &billing.ValidationError{})
			charge, err := s.UsageBasedAdapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: chargeID})
			s.Require().NoError(err)
			s.Require().True(lo.ContainsBy(charge.ValidationIssues, func(issue billing.ValidationIssue) bool {
				return issue.Code == billing.ErrInvoiceLineFeatureHasNoMeters.Code
			}))
			gathering, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
				Invoice: pending.Invoice.GetInvoiceID(),
				Expand:  billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
			})
			s.Require().NoError(err)
			s.Require().Len(gathering.Lines.OrEmpty(), 1)
			s.Equal(pending.Lines[0].ID, gathering.Lines.OrEmpty()[0].ID)

			// when: the missing dependency is repaired and collection is retried.
			s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, meters))
			s.MockStreamingConnector.AddSimpleEvent(meters[0].Key, 1, period.From.Add(time.Hour))
			s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)
			defer s.UsageBasedTestHandler.Reset()
			invoices, err := s.BillingService.InvoicePendingLines(ctx, input, billing.WithBypassCollectionAlignment())
			s.Require().NoError(err)

			// then: collection creates one invoice and clears the stale dependency issue.
			s.Require().Len(invoices, 1)
			charge, err = s.UsageBasedAdapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: chargeID})
			s.Require().NoError(err)
			s.False(lo.ContainsBy(charge.ValidationIssues, func(issue billing.ValidationIssue) bool {
				return issue.Code == billing.ErrInvoiceLineFeatureHasNoMeters.Code
			}))
		})
	}
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesResolvesStripeTaxCodeBeforeCreatingCharge() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-stripe-tax")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	line := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        servicePeriod,
		InvoiceAt:     servicePeriod.To,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "manual flat",
		Currency:      currencyx.FiatCode(USD),
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	line.Engine = billing.LineEngineTypeInvoice
	line.TaxConfig = &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_40010001"},
	}

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
		Lines:    billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{line}),
	})
	s.Require().NoError(err)
	s.Require().Len(result.Lines, 1)
	s.Require().NotNil(result.Lines[0].TaxConfig)
	s.Require().NotNil(result.Lines[0].TaxConfig.TaxCodeID)
	s.Require().NotNil(result.Lines[0].TaxConfig.Stripe)
	s.Equal("txcd_40010001", result.Lines[0].TaxConfig.Stripe.Code)

	resolved, err := s.TaxCodeEnv.Service.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: ns,
		AppType:   app.AppTypeStripe,
		TaxCode:   "txcd_40010001",
	})
	s.Require().NoError(err)
	s.Equal(resolved.ID, *result.Lines[0].TaxConfig.TaxCodeID)

	charge := s.mustGetChargeByID(meta.ChargeID{Namespace: ns, ID: lo.FromPtr(result.Lines[0].ChargeID)})
	flatFee, err := charge.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(resolved.ID, flatFee.Intent.GetTaxConfig().TaxCodeID)
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesCreatesChargeBackedGatheringLines() {
	defer s.FlatFeeTestHandler.Reset()
	s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
		return nil, nil
	}

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)

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
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			usageLine,
			flatLine,
		}),
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
				Lines:    billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{line}),
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

func (s *InvoicableChargesTestSuite) TestBillingCreatePendingInvoiceLinesBypassesFeatureMeterValidationPerLine() {
	// given:
	// - a meterless usage line trusted by subscription reconciliation
	// - an ordinary usage line with a valid feature meter
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("billing-create-pending-line-bypass-feature-meter")
	cust := s.CreateTestCustomer(ns, "test-subject")
	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID())

	meterlessFeatureKey := "meterless-pending-line-feature"
	_, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: ns,
		Name:      meterlessFeatureKey,
		Key:       meterlessFeatureKey,
	})
	s.Require().NoError(err)
	meteredFeature := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	meterlessLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: ns,
				Name:      "meterless usage",
			}),
			ManagedBy:     billing.ManuallyManagedLine,
			Currency:      currencyx.FiatCode(USD),
			ServicePeriod: servicePeriod,
			InvoiceAt:     servicePeriod.To,
			Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(2),
			})),
			FeatureKey: meterlessFeatureKey,
		},
	}
	meteredLine, err := meterlessLine.CloneForCreate(func(line *billing.GatheringLine) {
		line.Name = "metered usage"
		line.FeatureKey = meteredFeature.Feature.Key
	})
	s.Require().NoError(err)

	// when:
	// - both lines are created in one mixed batch and only the meterless line bypasses dependency validation
	result, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
		Lines: billing.CreatePendingInvoiceLines{
			{
				GatheringLine:                meterlessLine,
				BypassFeatureMeterValidation: true,
			},
			billing.NewCreatePendingInvoiceLine(meteredLine),
		},
	})

	// then:
	// - billing validates the ordinary line and persists both lines
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Len(result.Lines, 2)
	s.Equal(meterlessFeatureKey, result.Lines[0].FeatureKey)
	s.Equal(meteredFeature.Feature.Key, result.Lines[1].FeatureKey)

	// and:
	// - bypassing one line does not disable validation for another line in the same batch
	result, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
		Lines: billing.CreatePendingInvoiceLines{
			{
				GatheringLine:                meterlessLine,
				BypassFeatureMeterValidation: true,
			},
			billing.NewCreatePendingInvoiceLine(meterlessLine),
		},
	})
	s.Nil(result)
	s.Require().Error(err)
	issue := requireFeatureMeterValidationIssue(s.T(), err, billing.ErrInvoiceLineFeatureHasNoMeters.Code)
	s.Empty(issue.Path)
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
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

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
		Lines:    billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{line}),
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
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

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
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			zeroFlatLine,
		}),
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
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			systemLine,
		}),
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
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			subscriptionLine,
		}),
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "subscription is not allowed")
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRollsBackPartialChargeLineResults() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-partial-rollback")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)

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
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			usageLine,
			zeroFlatLine,
		}),
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

func (s *InvoicableChargesTestSuite) TestInvoiceCustomerChargesAPIUsesChargeEngine() {
	// given: credits are enabled, with a due charge and a future charge pending.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("manual-collection-api")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(), billingtest.WithManualApproval())
	cust := s.CreateTestCustomer(ns, "test-subject")
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.FreezeTime(period.To)
	defer clock.UnFreeze()

	allocationCalls := 0
	s.FlatFeeTestHandler.onAllocateCredits = func(_ context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
		allocationCalls++

		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.ServicePeriod,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
				Amount: alpacadecimal.NewFromInt(4),
			},
		}, nil
	}
	defer s.FlatFeeTestHandler.Reset()

	dueLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        period,
		InvoiceAt:     period.To,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "due charge",
		Currency:      currencyx.FiatCode(USD),
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	futureLine := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        period,
		InvoiceAt:     period.To.Add(time.Hour),
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "future charge",
		Currency:      currencyx.FiatCode(USD),
		PerUnitAmount: alpacadecimal.NewFromInt(10),
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
	pending, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(USD),
		Lines:    billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{dueLine, futureLine}),
	})
	s.Require().NoError(err)
	s.Require().NotNil(pending)
	s.Require().Len(pending.Lines, 2)
	s.Equal(billing.LineEngineTypeChargeFlatFee, pending.Lines[0].Engine)
	s.Require().NotNil(pending.Lines[0].ChargeID)

	// when: manually collecting through the v3 handler at the due line's cutoff.
	body, err := json.Marshal(api.InvoiceCustomerChargesInput{
		AsOf:                       lo.ToPtr(period.To),
		ProgressiveBillingOverride: lo.ToPtr(false),
	})
	s.Require().NoError(err)
	req := httptest.NewRequest(http.MethodPost, "/openmeter/customers/"+cust.ID+"/charges/invoice", bytes.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler := billinginvoices.New(func(context.Context) (string, error) { return ns, nil }, s.BillingService)
	handler.InvoiceCustomerCharges().With(cust.ID).ServeHTTP(res, req)

	// then: collection invokes the charge engine, applies credits, and retains the future line.
	s.Require().Equal(http.StatusCreated, res.Code, res.Body.String())
	var result api.InvoiceCustomerChargesResult
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &result))
	s.Require().Len(result.Invoices, 1)
	invoice, err := result.Invoices[0].AsBillingInvoiceStandard()
	s.Require().NoError(err)
	s.Equal("6", invoice.Totals.Total)
	s.Equal("4", invoice.Totals.CreditsTotal)
	s.Equal(1, allocationCalls)
	s.Require().NotNil(invoice.Lines)
	s.Require().Len(*invoice.Lines, 1)
	line, err := (*invoice.Lines)[0].AsBillingInvoiceStandardLine()
	s.Require().NoError(err)
	s.Require().NotNil(line.Charge)
	s.Equal(*pending.Lines[0].ChargeID, line.Charge.Id)
	s.Equal(lo.ToPtr(pending.Lines[0].ID), line.Id)

	charge := s.mustGetChargeByID(meta.ChargeID{Namespace: ns, ID: *pending.Lines[0].ChargeID})
	flatFeeCharge, err := charge.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Require().NotNil(flatFeeCharge.Realizations.CurrentRun)
	s.Equal(lo.ToPtr(invoice.Id), flatFeeCharge.Realizations.CurrentRun.InvoiceID)
	s.Equal(lo.ToPtr(pending.Lines[0].ID), flatFeeCharge.Realizations.CurrentRun.LineID)

	gathering, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
		Invoice: pending.Invoice.GetInvoiceID(),
		Expand:  billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
	})
	s.Require().NoError(err)
	s.Require().Len(gathering.Lines.OrEmpty(), 1)
	s.Equal(pending.Lines[1].ID, gathering.Lines.OrEmpty()[0].ID)
}
