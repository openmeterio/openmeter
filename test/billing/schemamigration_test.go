package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingstandardinvoicedetailedline"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SchemaMigrationTestSuite struct {
	BaseSuite
}

func TestSchemaMigration(t *testing.T) {
	suite.Run(t, new(SchemaMigrationTestSuite))
}

func (s *SchemaMigrationTestSuite) SetupSuite() {
	s.BaseSuite.setupSuite(SetupSuiteOptions{ForceAtlas: true})
}

func (s *SchemaMigrationTestSuite) TestSchemaLevel1Migration() {
	namespace := s.GetUniqueNamespace("ns-schema-migration")
	ctx := context.Background()

	const (
		lineNameDeletedDetailed = "Test item1"
		lineNameActiveDetailed  = "Test item2"
	)

	var (
		customerEntity *customer.Customer
		invoiceID      billing.InvoiceID

		deletedAtSet time.Time

		// Adapter snapshot (schema level 1 read-path)
		invoiceBeforeMigration billing.Invoice
	)

	s.Run("Given a customer and progressive billing profile exists", func() {
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithProgressiveBilling())

		customerEntity = s.CreateTestCustomer(namespace, "test-customer")
		s.NotNil(customerEntity)
	})

	var featureFlatPerUnit feature.Feature

	s.Run("Given a metered feature exists with usage", func() {
		meterSlug := "flat-per-unit"

		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
			{
				ManagedResource: models.ManagedResource{
					ID: ulid.Make().String(),
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "Flat per unit",
				},
				Key:           meterSlug,
				Aggregation:   meter.MeterAggregationSum,
				EventType:     "test",
				ValueProperty: lo.ToPtr("$.value"),
			},
		}))

		periodStart := time.Now().Add(-time.Hour)

		// Make sure the meter exists before the interesting event.
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
		// Register some usage.
		s.MockStreamingConnector.AddSimpleEvent(meterSlug, 10, periodStart.Add(time.Minute))

		featureFlatPerUnit = lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      meterSlug,
			Key:       meterSlug,
			MeterSlug: lo.ToPtr(meterSlug),
		}))
	})

	s.Run("Given schema level 1 invoice exists with amount discounts and a deleted detailed line", func() {
		periodStart := time.Now().Add(-time.Hour)
		periodEnd := time.Now().Add(time.Hour)

		_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.Line{
				{
					LineBase: billing.LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      lineNameDeletedDetailed,
						}),
						Period:    billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt: periodEnd,
						ManagedBy: billing.ManuallyManagedLine,
						Currency:  currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: featureFlatPerUnit.Key,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						}),
					},
				},
				{
					LineBase: billing.LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      lineNameActiveDetailed,
						}),
						Period:    billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt: periodEnd,
						ManagedBy: billing.ManuallyManagedLine,
						Currency:  currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: featureFlatPerUnit.Key,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						}),
					},
				},
			},
		})
		s.NoError(err)

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoiceID = invoices[0].InvoiceID()

		// Delete all detailed lines under the chosen parent by directly updating the schema-level-1 representation (billing_invoice_lines).
		deletedAtSet = clock.Now()
		n, err := s.markAllDetailedChildrenDeleted(ctx, namespace, invoiceID.ID, lineNameDeletedDetailed, deletedAtSet)
		s.NoError(err)
		s.Equal(n, 1)

		// Validate schema-level-1 using the adapter (read-path).
		invoiceBeforeMigration, err = s.BillingAdapter.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.InvoiceExpandAll.SetDeletedLines(true),
		})
		s.NoError(err)
		s.Len(invoiceBeforeMigration.Lines.OrEmpty(), 2)

		// One line has its only detailed line deleted -> adapter won't return deleted detailed lines.
		lineDeleted := s.getLineByName(invoiceBeforeMigration, lineNameDeletedDetailed)
		lineActive := s.getLineByName(invoiceBeforeMigration, lineNameActiveDetailed)

		s.Len(lineDeleted.DetailedLines, 0)
		s.Len(lineActive.DetailedLines, 1)
		s.GreaterOrEqual(len(lineActive.DetailedLines[0].AmountDiscounts), 1)
	})

	s.Run("When the write schema level is set to 2 and a lock is obtained on the customer", func() {
		s.NoError(s.BillingAdapter.SetInvoiceWriteSchemaLevel(ctx, 2))
		// Side-effect: migration happens due to the previous line.
		s.NoError(s.BillingAdapter.LockCustomerForUpdate(ctx, customerEntity.GetID()))
	})

	s.Run("Then the invoice is migrated and lines (incl detailed lines) match exactly", func() {
		invoiceAfter, err := s.BillingAdapter.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.InvoiceExpandAll.SetDeletedLines(true),
		})
		s.Require().NoError(err)

		// Invoice schema level updated (part of the invoice payload).
		s.Require().Equal(2, invoiceAfter.SchemaLevel)

		// Lines and their detailed lines match exactly (by name to avoid ordering assumptions).
		beforeDeleted := s.getLineByName(invoiceBeforeMigration, lineNameDeletedDetailed).WithoutSplitLineHierarchy().WithoutDBState()
		beforeActive := s.getLineByName(invoiceBeforeMigration, lineNameActiveDetailed).WithoutSplitLineHierarchy().WithoutDBState()
		afterDeleted := s.getLineByName(invoiceAfter, lineNameDeletedDetailed).WithoutSplitLineHierarchy().WithoutDBState()
		afterActive := s.getLineByName(invoiceAfter, lineNameActiveDetailed).WithoutSplitLineHierarchy().WithoutDBState()

		// Let's remove the DetailedLine's FeeLineConfigID as that's not existing in the schema-level-2 representation.
		beforeDeleted = s.withoutDetailedFeeLineConfigID(beforeDeleted)
		beforeActive = s.withoutDetailedFeeLineConfigID(beforeActive)

		s.Equal(beforeDeleted, afterDeleted)
		s.Equal(beforeActive, afterActive)

		// Detailed lines copied (2 input lines => 2 detailed fee lines; one is deleted).
		detailedLineCount, err := s.DBClient.BillingStandardInvoiceDetailedLine.Query().
			Where(billingstandardinvoicedetailedline.Namespace(namespace)).
			Where(billingstandardinvoicedetailedline.InvoiceID(invoiceID.ID)).
			Count(ctx)
		s.Require().NoError(err)
		s.Require().Equal(2, detailedLineCount)

		// Deleted detailed line is copied (find by deleted_at we set).
		deletedCopiedCount, err := s.DBClient.BillingStandardInvoiceDetailedLine.Query().
			Where(billingstandardinvoicedetailedline.Namespace(namespace)).
			Where(billingstandardinvoicedetailedline.InvoiceID(invoiceID.ID)).
			Where(billingstandardinvoicedetailedline.DeletedAtEQ(deletedAtSet)).
			Count(ctx)
		s.Require().NoError(err)
		s.Require().Equal(1, deletedCopiedCount)
	})
}

func (s *SchemaMigrationTestSuite) markAllDetailedChildrenDeleted(ctx context.Context, ns string, invoiceID string, parentLineName string, deletedAt time.Time) (int, error) {
	// Find the parent (top-level) invoice line by name.
	parent, err := s.DBClient.BillingInvoiceLine.Query().
		Where(billinginvoiceline.Namespace(ns)).
		Where(billinginvoiceline.InvoiceID(invoiceID)).
		Where(billinginvoiceline.NameEQ(parentLineName)).
		Where(billinginvoiceline.StatusEQ(billing.InvoiceLineStatusValid)).
		First(ctx)
	if err != nil {
		return 0, fmt.Errorf("finding parent invoice line %q: %w", parentLineName, err)
	}

	// Mark ALL child detailed lines under that parent as deleted.
	n, err := s.DBClient.BillingInvoiceLine.Update().
		Where(billinginvoiceline.Namespace(ns)).
		Where(billinginvoiceline.InvoiceID(invoiceID)).
		Where(billinginvoiceline.ParentLineIDEQ(parent.ID)).
		Where(billinginvoiceline.StatusEQ(billing.InvoiceLineStatusDetailed)).
		Where(billinginvoiceline.DeletedAtIsNil()).
		SetDeletedAt(deletedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("setting deleted_at on detailed lines under parent %q: %w", parentLineName, err)
	}

	return n, nil
}

func (s *SchemaMigrationTestSuite) withoutDetailedFeeLineConfigID(in *billing.Line) *billing.Line {
	if in == nil {
		return nil
	}

	out := in.Clone()
	for i := range out.DetailedLines {
		out.DetailedLines[i].FeeLineConfigID = ""
	}
	return out
}

func (s *SchemaMigrationTestSuite) getLineByName(inv billing.Invoice, name string) *billing.Line {
	lines := inv.Lines.OrEmpty()
	for _, l := range lines {
		if l.Name == name {
			return l
		}
	}
	s.FailNowf("line not found", "invoice does not contain a line with name %q", name)
	return nil
}
