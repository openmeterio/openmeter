package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

type CollectionTestSuite struct {
	BaseSuite
}

func TestCollection(t *testing.T) {
	suite.Run(t, new(CollectionTestSuite))
}

type collectionNSResult struct {
	TestFeature
	customer *customer.Customer
}

func (s *CollectionTestSuite) setupNS(ctx context.Context, namespace string) collectionNSResult {
	s.T().Helper()

	s.InstallSandboxApp(s.T(), namespace)

	customer := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customer)

	apiRequestsTotalFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)

	minimalCreateProfileInput := MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace
	minimalCreateProfileInput.WorkflowConfig.Invoicing.ProgressiveBilling = true
	minimalCreateProfileInput.WorkflowConfig.Collection.Interval = isodate.MustParse(s.T(), "PT1H")

	_, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
	s.NoError(err)

	return collectionNSResult{
		TestFeature: apiRequestsTotalFeature,
		customer:    customer,
	}
}

func (s *CollectionTestSuite) TestCollectionFlow() {
	namespace := "ns-collection-flow"
	ctx := context.Background()

	res := s.setupNS(ctx, namespace)
	defer res.Cleanup()

	customer := res.customer
	apiRequestsTotalFeature := res.TestFeature

	periodStart := lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))
	periodEnd := periodStart.Add(time.Hour * 12)
	period2End := periodStart.Add(time.Hour * 24)

	clock.SetTime(periodStart)
	defer clock.ResetTime()

	// Given a profile with subscription aligned collection
	// When a gathering invoice have multiple lines with different billing periods
	// Then the collection_at should be set to the min of the invoice_at of the lines

	var gatheringInvoiceID billing.InvoiceID
	s.Run("validate collection_at calculation", func() {
		res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: customer.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.Line{
				{
					LineBase: billing.LineBase{
						Period:    billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt: periodEnd,
						ManagedBy: billing.ManuallyManagedLine,
						Type:      billing.InvoiceLineTypeUsageBased,
						Name:      "UBP - unit",
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: apiRequestsTotalFeature.Feature.Key,
						Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
					},
				},
				{
					LineBase: billing.LineBase{
						Period:    billing.Period{Start: periodStart, End: period2End},
						InvoiceAt: period2End,
						ManagedBy: billing.ManuallyManagedLine,
						Type:      billing.InvoiceLineTypeUsageBased,
						Name:      "UBP - volume",
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: apiRequestsTotalFeature.Feature.Key,
						Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
							Mode: productcatalog.VolumeTieredPrice,
							Tiers: []productcatalog.PriceTier{
								{
									UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
									UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromFloat(1)},
								},
								{
									UpToAmount: nil,
									UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromFloat(0.5)},
								},
							},
						}),
					},
				},
			},
		})
		s.NoError(err)
		s.Len(res.Lines, 2)

		gatheringInvoiceID = res.Invoice.InvoiceID()

		// Validate collection_at calculation
		s.NotNil(res.Invoice.CollectionAt)
		s.Equal(periodEnd, *res.Invoice.CollectionAt, "collection_at should be the min of the invoice_at of the lines")
	})

	// Given a gatherting invoice exists
	// When fetching the fully expanded gathering invoice
	// Then collection at is properly returned

	s.Run("validate collection_at for expanded gathering invoice", func() {
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand: billing.InvoiceExpand{
				Lines:                       true,
				RecalculateGatheringInvoice: true,
			},
		})
		s.NoError(err)

		s.NotNil(gatheringInvoice.CollectionAt)
		s.Equal(periodEnd, *gatheringInvoice.CollectionAt, "collection_at should be the min of the invoice_at of the lines")
	})

	s.NotEmpty(gatheringInvoiceID)

	// Given the previous gathering invoice exists
	// When:
	// - Clock is periodEnd + 30min
	// - A pending invoice has been created
	// - 1 usage is record at periodStart + 30min
	// Then:
	// - The collection at of the invoice is periodEnd + 1hr
	// - The invoice should wait for collection
	// - The invoice should not have trigger_next available
	// - The invoice should have correct totals based on the event (total=$2)

	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 1, periodStart.Add(time.Minute*30))
	defer s.MockStreamingConnector.Reset()

	var invoiceID billing.InvoiceID
	clock.SetTime(period2End.Add(time.Minute * 30))
	s.Run("validate collection_at for pending invoice", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice := invoices[0]

		s.NotNil(invoice.CollectionAt)
		s.Nil(invoice.QuantitySnapshotedAt)
		s.Equal(period2End.Add(time.Hour), *invoice.CollectionAt, "collection_at should be periodEnd + 1hr")

		s.Equal(billing.InvoiceStatusDraftWaitingForCollection, invoice.Status)
		s.Nil(invoice.StatusDetails.AvailableActions.Advance)

		// total should be $2
		s.Equal(float64(2), invoice.Totals.Amount.InexactFloat64())

		invoiceID = invoice.InvoiceID()
	})

	// Given the draft invoice is in waiting for collection state
	// When:
	// - Clock is period2End + 1hr
	// - A new usage of 2 is recorded at periodStart + 35min (late event)
	// Then:
	// - The invoice should be advancable
	// - The invoice should be in waiting for approval state
	// - The invoice should have correct totals based on the event (total=$2+$4)

	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 2, periodStart.Add(time.Minute*35))

	clock.SetTime(period2End.Add(time.Hour))
	s.Run("validate invoice is advancable", func() {
		invoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoiceID,
			Expand: billing.InvoiceExpand{
				Lines: true,
			},
		})
		s.NoError(err)

		s.Equal(billing.InvoiceStatusDraftWaitingForCollection, invoice.Status)
		s.NotNil(invoice.StatusDetails.AvailableActions.Advance)

		// advancement should work
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoiceID)
		s.NoError(err)
		s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.NotNil(invoice.QuantitySnapshotedAt)
		s.True(!invoice.QuantitySnapshotedAt.After(clock.Now()), "quantity should be snapshoted before now()")

		// total should be $6 (snapshot occurred)
		s.Equal(float64(6), invoice.Totals.Amount.InexactFloat64())
	})
}

func (s *CollectionTestSuite) TestCollectionFlowWithFlatFeeOnly() {
	namespace := "ns-collection-flow-flat-fee"
	ctx := context.Background()

	// Given a gathering invoice with a flat fee only line
	// When the invoice is created
	// Then the freshly created invoice should skip the collection period

	res := s.setupNS(ctx, namespace)
	defer res.Cleanup()

	customer := res.customer

	periodStart := lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))
	periodEnd := periodStart.Add(time.Hour * 12)

	clock.SetTime(periodStart)
	defer clock.ResetTime()

	// Given
	pendingLineResult, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customer.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []*billing.Line{
			billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
				Period:    billing.Period{Start: periodStart, End: periodEnd},
				InvoiceAt: periodStart,
				Name:      "Flat fee",

				PerUnitAmount: alpacadecimal.NewFromFloat(10),
				Quantity:      alpacadecimal.NewFromFloat(1),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		},
	})
	s.NoError(err)
	s.Len(pendingLineResult.Lines, 1)
	s.NotNil(pendingLineResult.Invoice.CollectionAt)

	// When
	clock.SetTime(periodStart.Add(time.Hour * 1))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]

	// Then
	s.NotNil(invoice.CollectionAt)
	s.NotNil(invoice.QuantitySnapshotedAt)
	s.Equal(invoice.CreatedAt, *invoice.CollectionAt)
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
}

func (s *CollectionTestSuite) TestCollectionFlowWithFlatFeeEditing() {
	namespace := "ns-collection-flow-flat-fee-editing"
	ctx := context.Background()

	res := s.setupNS(ctx, namespace)
	defer res.Cleanup()

	customer := res.customer
	apiRequestsTotalFeature := res.TestFeature

	periodStart := lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))
	periodEnd := periodStart.Add(time.Hour * 12)

	clock.SetTime(periodStart)
	defer clock.ResetTime()

	// Given an invoice with UBP, collection done, in waiting for auto approval state
	// When a flat fee is added
	// Then the invoice:
	// - should reach waiting for auto approval state again
	// - there should be no new snapshot happening (to prevent suprises, later we can implement late events handling)

	// Given
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 1, periodStart.Add(time.Minute*30))
	defer s.MockStreamingConnector.Reset()

	pendingLineResult, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customer.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []*billing.Line{
			{
				LineBase: billing.LineBase{
					Period:    billing.Period{Start: periodStart, End: periodEnd},
					InvoiceAt: periodEnd,
					ManagedBy: billing.ManuallyManagedLine,
					Type:      billing.InvoiceLineTypeUsageBased,
					Name:      "UBP - unit",
				},
				UsageBased: &billing.UsageBasedLine{
					FeatureKey: apiRequestsTotalFeature.Feature.Key,
					Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				},
			},
		},
	})
	s.NoError(err)
	s.Len(pendingLineResult.Lines, 1)
	s.NotNil(pendingLineResult.Invoice.CollectionAt)

	clock.SetTime(periodEnd.Add(time.Hour * 1))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	s.Equal(float64(1), invoice.Totals.Amount.InexactFloat64())

	previousSnapshot := *invoice.QuantitySnapshotedAt
	s.NotNil(previousSnapshot)

	// When adding a flat fee (in arrears)
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 1, periodStart.Add(time.Minute*35))

	invoice, err = s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: invoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			linePeriod := billing.Period{
				Start: periodEnd.Add(time.Hour * 1),
				End:   periodEnd.Add(time.Hour * 2),
			}

			invoice.Lines.Append(
				billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
					Namespace: namespace,
					Currency:  currencyx.Code(currency.USD),
					InvoiceID: invoice.ID,
					Period:    linePeriod,
					InvoiceAt: linePeriod.End,
					Name:      "Flat fee",

					PerUnitAmount: alpacadecimal.NewFromFloat(10),
					Quantity:      alpacadecimal.NewFromFloat(1),
					PaymentTerm:   productcatalog.InArrearsPaymentTerm,
				}),
			)
			return nil
		},
	})
	s.NoError(err)

	// Then
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)

	// No new snapshot should happen
	s.Equal(float64(11), invoice.Totals.Amount.InexactFloat64()) // Event at periodStart + 35min is ignored
	s.NotNil(invoice.QuantitySnapshotedAt)
	s.Equal(previousSnapshot, *invoice.QuantitySnapshotedAt)
}

func (s *CollectionTestSuite) TestCollectionFlowWithUBPEditingExtendingCollectionPeriod() {
	namespace := "ns-collection-flow-ubp-editing-extending-collection-period"
	ctx := context.Background()

	res := s.setupNS(ctx, namespace)
	defer res.Cleanup()

	customer := res.customer
	apiRequestsTotalFeature := res.TestFeature

	periodStart := lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))
	periodEnd := periodStart.Add(time.Hour * 12)

	clock.SetTime(periodStart)
	defer clock.ResetTime()

	// Given an invoice with UBP, collection done, in waiting for auto approval state
	// When:
	// - a UBP is edited, that would require extending the collection period
	// Then the invoice:
	// - should wait for collection again

	// Given
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 1, periodStart.Add(time.Minute*30))
	defer s.MockStreamingConnector.Reset()

	pendingLineResult, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customer.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []*billing.Line{
			{
				LineBase: billing.LineBase{
					Period:    billing.Period{Start: periodStart, End: periodEnd},
					InvoiceAt: periodEnd,
					ManagedBy: billing.ManuallyManagedLine,
					Type:      billing.InvoiceLineTypeUsageBased,
					Name:      "UBP - unit",
				},
				UsageBased: &billing.UsageBasedLine{
					FeatureKey: apiRequestsTotalFeature.Feature.Key,
					Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				},
			},
		},
	})
	s.NoError(err)
	s.Len(pendingLineResult.Lines, 1)
	s.NotNil(pendingLineResult.Invoice.CollectionAt)

	clock.SetTime(periodEnd.Add(time.Hour * 1))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	s.Equal(float64(1), invoice.Totals.Amount.InexactFloat64())

	previousSnapshot := *invoice.QuantitySnapshotedAt
	s.NotNil(previousSnapshot)

	// When adding an UBP line with a new billing period
	newLinePeriod := billing.Period{
		Start: lo.Must(time.Parse(time.RFC3339, "2025-01-02T00:00:00Z")),
		End:   lo.Must(time.Parse(time.RFC3339, "2025-01-03T00:00:00Z")),
	}
	s.Run("adding a new line extends the collection period", func() {
		invoice, err = s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
			Invoice: invoice.InvoiceID(),
			EditFn: func(invoice *billing.Invoice) error {
				invoice.Lines.Append(&billing.Line{
					LineBase: billing.LineBase{
						Namespace: namespace,
						Currency:  currencyx.Code(currency.USD),
						InvoiceID: invoice.ID,
						Status:    billing.InvoiceLineStatusValid,
						Period:    newLinePeriod,
						InvoiceAt: newLinePeriod.End,
						ManagedBy: billing.ManuallyManagedLine,
						Type:      billing.InvoiceLineTypeUsageBased,
						Name:      "UBP - unit - new",
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: apiRequestsTotalFeature.Feature.Key,
						Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(3)}),

						// Note: this emulates a per line quantity snapshot, that would be done by a normal edit flow
						Quantity:              lo.ToPtr(alpacadecimal.NewFromFloat(0)),
						PreLinePeriodQuantity: lo.ToPtr(alpacadecimal.NewFromFloat(0)),
					},
				})
				return nil
			},
		})
		s.NoError(err)

		// Then
		s.Equal(billing.InvoiceStatusDraftWaitingForCollection, invoice.Status)

		// No new snapshot should happen
		s.Equal(float64(1), invoice.Totals.Amount.InexactFloat64(), "no new total is registered")
		s.NotNil(invoice.QuantitySnapshotedAt, "snapshot should be set")
		s.Equal(previousSnapshot, *invoice.QuantitySnapshotedAt, "no new snapshot should happen")
	})

	// When:
	// - the invoice is advancable
	// - a new event is recorded
	// Then:
	// - the invoice should be in waiting for approval state
	// - the invoice should have updated snapshots, quantity snapshoted at and totals

	s.Run("advancing the invoice updates the snapshot", func() {
		clock.SetTime(newLinePeriod.End.Add(time.Hour))
		s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 1, newLinePeriod.Start.Add(time.Minute*30))

		invoice, err = s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.InvoiceID(),
			Expand: billing.InvoiceExpand{
				Lines: true,
			},
		})
		s.NoError(err)

		s.Equal(billing.InvoiceStatusDraftWaitingForCollection, invoice.Status)

		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.InvoiceID())
		s.NoError(err)

		s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.NotNil(invoice.QuantitySnapshotedAt)
		s.Equal(float64(4), invoice.Totals.Amount.InexactFloat64())
		s.NotEqual(previousSnapshot, *invoice.QuantitySnapshotedAt)
		s.True(!invoice.QuantitySnapshotedAt.Before(newLinePeriod.End), "quantity should be snapshoted after the new line period")
	})
}
