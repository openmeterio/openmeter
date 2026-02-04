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
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	customer := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customer)

	apiRequestsTotalFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(),
		WithProgressiveBilling(),
		WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
	)

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
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name: "UBP - unit",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     periodEnd,
						ManagedBy:     billing.ManuallyManagedLine,
						FeatureKey:    apiRequestsTotalFeature.Feature.Key,
						Price:         lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)})),
					},
				},
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name: "UBP - volume",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: period2End},
						InvoiceAt:     period2End,
						ManagedBy:     billing.ManuallyManagedLine,
						FeatureKey:    apiRequestsTotalFeature.Feature.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.TieredPrice{
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
						})),
					},
				},
			},
		})
		s.NoError(err)
		s.Len(res.Lines, 2)

		gatheringInvoiceID = res.Invoice.InvoiceID()

		// Validate collection_at calculation
		s.NotNil(res.Invoice.NextCollectionAt)
		s.Equal(periodEnd, res.Invoice.NextCollectionAt, "collection_at should be the min of the invoice_at of the lines")
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

		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
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

		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
		s.NotNil(invoice.StatusDetails.AvailableActions.Advance)

		// advancement should work
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.NotNil(invoice.QuantitySnapshotedAt)
		s.True(!invoice.QuantitySnapshotedAt.After(clock.Now()), "quantity should be snapshoted before now()")

		// total should be $6 (snapshot occurred)
		s.Equal(float64(6), invoice.Totals.Amount.InexactFloat64())
	})
}

func (s *CollectionTestSuite) TestCollectionFlowWithFlatFeeOnly() {
	periodStart := lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))
	periodEnd := periodStart.Add(time.Hour * 12)

	// TODO[later]: When flat_fee on invoice is deprecated, we can remove the multiple testcase approach here and test UBP only
	tcs := []struct {
		name      string
		namespace string
		line      billing.GatheringLine
	}{
		{
			name:      "flat fee only",
			namespace: "ns-collection-flow-flat-fee",
			line: billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Period:    billing.Period{Start: periodStart, End: periodEnd},
				InvoiceAt: periodStart,
				Name:      "Flat fee",

				PerUnitAmount: alpacadecimal.NewFromFloat(10),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		},
		{
			name:      "ubp flat fee only",
			namespace: "ns-collection-flow-ubp-flat-fee",
			line: billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Period:    billing.Period{Start: periodStart, End: periodEnd},
				InvoiceAt: periodStart,
				Name:      "Flat fee",

				PerUnitAmount: alpacadecimal.NewFromFloat(10),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		},
	}

	for _, tc := range tcs {
		s.Run(tc.name, func() {
			namespace := tc.namespace
			ctx := context.Background()

			// Given a gathering invoice with a flat fee only line
			// When the invoice is created
			// Then the freshly created invoice should skip the collection period

			res := s.setupNS(ctx, namespace)
			defer res.Cleanup()

			customer := res.customer

			clock.SetTime(periodStart)
			defer clock.ResetTime()

			// Given
			pendingLineResult, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: customer.GetID(),
				Currency: currencyx.Code(currency.USD),
				Lines:    []billing.GatheringLine{tc.line},
			})
			s.NoError(err)
			s.Len(pendingLineResult.Lines, 1)
			s.NotNil(pendingLineResult.Invoice.NextCollectionAt)

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
			s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		})
	}
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
		Lines: []billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: "UBP - unit"}),
					ServicePeriod:   timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
					InvoiceAt:       periodEnd,
					ManagedBy:       billing.ManuallyManagedLine,
					FeatureKey:      apiRequestsTotalFeature.Feature.Key,
					Price:           *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				},
			},
		},
	})
	s.NoError(err)
	s.Len(pendingLineResult.Lines, 1)
	s.NotNil(pendingLineResult.Invoice.NextCollectionAt)

	clock.SetTime(periodEnd.Add(time.Hour * 1))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	s.Equal(float64(1), invoice.Totals.Amount.InexactFloat64())

	previousSnapshot := *invoice.QuantitySnapshotedAt
	s.NotNil(previousSnapshot)

	// When adding a flat fee (in arrears)
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalFeature.Feature.Key, 1, periodStart.Add(time.Minute*35))

	invoice, err = s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: invoice.InvoiceID(),
		EditFn: func(invoice *billing.StandardInvoice) error {
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
					PaymentTerm:   productcatalog.InArrearsPaymentTerm,
				}),
			)
			return nil
		},
	})
	s.NoError(err)

	// Then
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

	// No new snapshot should happen
	s.Equal(float64(11), invoice.Totals.Amount.InexactFloat64()) // Event at periodStart + 35min is ignored
	s.NotNil(invoice.QuantitySnapshotedAt)
	s.Equal(previousSnapshot, *invoice.QuantitySnapshotedAt)
}

func (s *CollectionTestSuite) TestAnchoredAlignment_SetsCollectionAtToNextAnchor() {
	namespace := "ns-anchored-collection-at"
	ctx := context.Background()
	defer clock.ResetTime()

	now := lo.Must(time.Parse(time.RFC3339, "2025-06-15T12:00:00Z"))
	clock.SetTime(now)

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	// Billing profile with anchored alignment to first of month
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Collection.Alignment = billing.AlignmentKindAnchored
		profile.WorkflowConfig.Collection.AnchoredAlignmentDetail = lo.ToPtr(billing.AnchoredAlignmentDetail{
			Interval: lo.Must(datetime.ISODurationString("P1M").Parse()),
			Anchor:   time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		})
	}))

	// Create customer
	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name:           "Test Customer",
			BillingAddress: &models.Address{Country: lo.ToPtr(models.CountryCode("US"))},
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test-subject-1"},
			},
		},
	})
	s.Require().NoError(err)

	// Create a minimal pending usage-based line that invoices at end of day
	periodStart := now
	periodEnd := now.Add(12 * time.Hour)
	_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customerEntity.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: "UBP - unit"}),
					ServicePeriod:   timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
					InvoiceAt:       periodEnd,
					ManagedBy:       billing.ManuallyManagedLine,
					FeatureKey:      s.SetupApiRequestsTotalFeature(ctx, namespace).Feature.Key,
					Price:           *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				},
			},
		},
	})
	s.Require().NoError(err)

	// Let's advance time
	clock.SetTime(periodEnd.Add(time.Hour * 1))
	defer clock.ResetTime()

	// Create standard invoice from pending lines; collectionAt should be next anchor (1st of next month)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	inv := invoices[0]
	s.Require().NotNil(inv.CollectionAt)
	nextAnchor := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	s.Equal(nextAnchor, *inv.CollectionAt)

	// Assert the workflow collection alignment is snapshotted on the invoice
	s.Equal(billing.AlignmentKindAnchored, inv.Workflow.Config.Collection.Alignment)
	s.Require().NotNil(inv.Workflow.Config.Collection.AnchoredAlignmentDetail)
	s.Equal("P1M", inv.Workflow.Config.Collection.AnchoredAlignmentDetail.Interval.String())
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
		Lines: []billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: "UBP - unit"}),
					ServicePeriod:   timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
					InvoiceAt:       periodEnd,
					ManagedBy:       billing.ManuallyManagedLine,
					FeatureKey:      apiRequestsTotalFeature.Feature.Key,
					Price:           *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				},
			},
		},
	})
	s.NoError(err)
	s.Len(pendingLineResult.Lines, 1)
	s.NotNil(pendingLineResult.Invoice.NextCollectionAt)

	clock.SetTime(periodEnd.Add(time.Hour * 1))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
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
			EditFn: func(invoice *billing.StandardInvoice) error {
				invoice.Lines.Append(&billing.StandardLine{
					StandardLineBase: billing.StandardLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "UBP - unit - new",
						}),
						Currency:  currencyx.Code(currency.USD),
						InvoiceID: invoice.ID,
						Period:    newLinePeriod,
						InvoiceAt: newLinePeriod.End,
						ManagedBy: billing.ManuallyManagedLine,
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: apiRequestsTotalFeature.Feature.Key,
						Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(3)}),

						// Note: this emulates a per line quantity snapshot, that would be done by a normal edit flow
						Quantity:                     lo.ToPtr(alpacadecimal.NewFromFloat(0)),
						MeteredQuantity:              lo.ToPtr(alpacadecimal.NewFromFloat(0)),
						PreLinePeriodQuantity:        lo.ToPtr(alpacadecimal.NewFromFloat(0)),
						MeteredPreLinePeriodQuantity: lo.ToPtr(alpacadecimal.NewFromFloat(0)),
					},
				})
				return nil
			},
		})
		s.NoError(err)

		// Then
		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)

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

		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)

		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.InvoiceID())
		s.NoError(err)

		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.NotNil(invoice.QuantitySnapshotedAt)
		s.Equal(float64(4), invoice.Totals.Amount.InexactFloat64())
		s.NotEqual(previousSnapshot, *invoice.QuantitySnapshotedAt)
		s.True(!invoice.QuantitySnapshotedAt.Before(newLinePeriod.End), "quantity should be snapshoted after the new line period")
	})
}
