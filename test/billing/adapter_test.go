package billing

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesearchv1"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type BillingAdapterTestSuite struct {
	BaseSuite
	SubscriptionMixin
}

func TestBillingAdapter(t *testing.T) {
	suite.Run(t, new(BillingAdapterTestSuite))
}

func (s *BillingAdapterTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.SubscriptionMixin.SetupSuite(s.T(), s.GetSubscriptionMixInDependencies())
}

func (s *BillingAdapterTestSuite) setupInvoice(ctx context.Context, ns string) *billing.StandardInvoice {
	s.T().Helper()
	// Given we have a customer
	customerEntity := s.CreateTestCustomer(ns, "test")
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	// Given we have a profile
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	// Given we have an invoice
	invoice, err := s.BillingAdapter.CreateInvoice(ctx, billing.CreateInvoiceAdapterInput{
		Namespace: ns,
		Customer:  *customerEntity,

		Number:   "INV-123",
		Currency: currencyx.Code(currency.USD),
		Status:   billing.StandardInvoiceStatusGathering,

		Profile:  *profile,
		IssuedAt: time.Now(),

		Type: billing.InvoiceTypeStandard,
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), invoice)

	return &invoice
}

func (s *BillingAdapterTestSuite) TestReadInvoicesAcrossInvoiceTables() {
	// given:
	// - one standard invoice and one legacy gathering invoice in billing_invoices
	// - one gathering invoice and line in the dedicated tables
	// when:
	// - generic listing and type-specific reads are requested
	// then:
	// - generic listing follows view order and hydrates all storage locations
	// - gathering invoice reads use the storage table selected by the view
	ctx := s.T().Context()
	standardNamespace := s.GetUniqueNamespace("list-invoices-standard")
	gatheringNamespace := s.GetUniqueNamespace("list-invoices-gathering")

	standardInvoice := s.setupInvoice(ctx, standardNamespace)
	_, err := s.DBClient.BillingInvoice.UpdateOneID(standardInvoice.ID).
		SetStatus(billing.StandardInvoiceStatusDraftCreated).
		Save(ctx)
	require.NoError(s.T(), err)

	legacyGatheringInvoice := s.setupInvoice(ctx, gatheringNamespace)
	dedicatedInvoiceID := ulid.Make().String()
	dedicatedLineID := ulid.Make().String()
	subscriptionStart := clock.Now().UTC()
	dedicatedCreatedAt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	createdPlan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: gatheringNamespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Dedicated gathering invoice test plan",
				Key:            "dedicated-gathering-invoice-test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name: "default",
						Key:  "default",
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee",
								Name: "Flat fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(10),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, gatheringNamespace, productcatalogsubscription.PlanRefInput{
		Key:     createdPlan.Key,
		Version: lo.ToPtr(createdPlan.Version),
	})
	require.NoError(s.T(), err)

	subscriptionView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(subscriptionStart),
			},
		},
		Namespace:  gatheringNamespace,
		CustomerID: legacyGatheringInvoice.GetCustomerID().ID,
	}, subscriptionPlan)
	require.NoError(s.T(), err)
	subscriptionID := subscriptionView.Subscription.ID
	require.Len(s.T(), subscriptionView.Phases, 1)
	subscriptionPhase := subscriptionView.Phases[0].SubscriptionPhase
	subscriptionItems := subscriptionView.Phases[0].ItemsByKey["flat-fee"]
	require.Len(s.T(), subscriptionItems, 1)
	subscriptionItem := subscriptionItems[0].SubscriptionItem
	require.Equal(s.T(), subscriptionPhase.ID, subscriptionItem.PhaseId)

	billingPeriod, err := subscriptionView.Spec.GetAlignedBillingPeriodAt(subscriptionStart)
	require.NoError(s.T(), err)
	require.True(s.T(), billingPeriod.Contains(subscriptionItem.ActiveFrom))
	servicePeriodStart := billingPeriod.From
	servicePeriodEnd := billingPeriod.To

	_, err = s.DBClient.BillingGatheringInvoice.Create().
		SetID(dedicatedInvoiceID).
		SetNamespace(gatheringNamespace).
		SetName("Dedicated gathering invoice").
		SetNumber("GATHERING-DEDICATED").
		SetCustomerID(legacyGatheringInvoice.GetCustomerID().ID).
		SetCurrency(legacyGatheringInvoice.Currency).
		SetServicePeriodStart(servicePeriodStart).
		SetServicePeriodEnd(servicePeriodEnd).
		SetNextCollectionAt(servicePeriodEnd).
		SetCreatedAt(dedicatedCreatedAt).
		SetUpdatedAt(dedicatedCreatedAt).
		SetSchemaLevel(2).
		Save(ctx)
	require.NoError(s.T(), err)

	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(10),
	})
	_, err = s.DBClient.BillingGatheringInvoiceLine.Create().
		SetID(dedicatedLineID).
		SetNamespace(gatheringNamespace).
		SetName("Dedicated gathering line").
		SetCurrency(legacyGatheringInvoice.Currency).
		SetServicePeriodStart(servicePeriodStart).
		SetServicePeriodEnd(servicePeriodEnd).
		SetPriceType(price.Type()).
		SetFeatureKey("dedicated-feature").
		SetPrice(price).
		SetInvoiceID(dedicatedInvoiceID).
		SetInvoiceAt(servicePeriodEnd).
		SetManagedBy(billing.SystemManagedLine).
		SetEngine(billing.LineEngineTypeInvoice).
		SetSubscriptionID(subscriptionID).
		SetSubscriptionPhaseID(subscriptionPhase.ID).
		SetSubscriptionItemID(subscriptionItem.ID).
		Save(ctx)
	require.NoError(s.T(), err)

	s.Run("search view identifies each invoice storage table", func() {
		ctx := s.T().Context()
		searchRows, err := s.DBClient.BillingInvoiceSearchV1.Query().
			Where(billinginvoicesearchv1.IDIn(standardInvoice.ID, legacyGatheringInvoice.ID, dedicatedInvoiceID)).
			All(ctx)
		require.NoError(s.T(), err)
		storageTableByInvoiceID := lo.SliceToMap(searchRows, func(row *db.BillingInvoiceSearchV1) (string, billinginvoicesearchv1.StorageTable) {
			return row.ID, row.StorageTable
		})
		require.Equal(s.T(), billinginvoicesearchv1.StorageTableBillingInvoice, storageTableByInvoiceID[standardInvoice.ID])
		require.Equal(s.T(), billinginvoicesearchv1.StorageTableBillingInvoice, storageTableByInvoiceID[legacyGatheringInvoice.ID])
		require.Equal(s.T(), billinginvoicesearchv1.StorageTableBillingGatheringInvoice, storageTableByInvoiceID[dedicatedInvoiceID])
	})

	s.Run("ListInvoices hydrates standard and gathering invoices across storage tables", func() {
		ctx := s.T().Context()
		listed, err := s.BillingAdapter.ListInvoices(ctx, billing.ListInvoicesAdapterInput{
			Namespaces: []string{standardNamespace, gatheringNamespace},
			IDs:        []string{standardInvoice.ID, legacyGatheringInvoice.ID, dedicatedInvoiceID},
			Expand:     billing.InvoiceExpands{billing.InvoiceExpandLines},
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), 3, listed.TotalCount)
		require.Len(s.T(), listed.Items, 3)
		firstInvoice, err := listed.Items[0].AsGenericInvoice()
		require.NoError(s.T(), err)
		require.Equal(s.T(), dedicatedInvoiceID, firstInvoice.GetID())

		listedByID := make(map[string]billing.Invoice, len(listed.Items))
		for _, invoice := range listed.Items {
			genericInvoice, err := invoice.AsGenericInvoice()
			require.NoError(s.T(), err)
			listedByID[genericInvoice.GetID()] = invoice
		}

		standard, err := listedByID[standardInvoice.ID].AsStandardInvoice()
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.StandardInvoiceStatusDraftCreated, standard.Status)

		legacyGathering, err := listedByID[legacyGatheringInvoice.ID].AsGatheringInvoice()
		require.NoError(s.T(), err)
		require.Equal(s.T(), legacyGatheringInvoice.ID, legacyGathering.ID)

		dedicatedGathering, err := listedByID[dedicatedInvoiceID].AsGatheringInvoice()
		require.NoError(s.T(), err)
		require.Equal(s.T(), "GATHERING-DEDICATED", dedicatedGathering.Number)
		require.Equal(s.T(), servicePeriodStart, dedicatedGathering.ServicePeriod.From)
		require.Equal(s.T(), servicePeriodEnd, dedicatedGathering.ServicePeriod.To)
		require.Len(s.T(), dedicatedGathering.Lines.OrEmpty(), 1)
		require.Equal(s.T(), dedicatedLineID, dedicatedGathering.Lines.OrEmpty()[0].ID)
		require.NotNil(s.T(), dedicatedGathering.Lines.OrEmpty()[0].DBState)
	})

	s.Run("GetGatheringInvoiceById loads a legacy gathering invoice", func() {
		legacyGatheringByID, err := s.BillingAdapter.GetGatheringInvoiceById(s.T().Context(), billing.GetGatheringInvoiceByIdInput{
			Invoice: legacyGatheringInvoice.GetInvoiceID(),
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), legacyGatheringInvoice.ID, legacyGatheringByID.ID)
	})

	s.Run("GetGatheringInvoiceById loads a dedicated gathering invoice with lines", func() {
		dedicatedGatheringByID, err := s.BillingAdapter.GetGatheringInvoiceById(s.T().Context(), billing.GetGatheringInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: gatheringNamespace,
				ID:        dedicatedInvoiceID,
			},
			Expand: billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), dedicatedInvoiceID, dedicatedGatheringByID.ID)
		require.Len(s.T(), dedicatedGatheringByID.Lines.OrEmpty(), 1)
		require.Equal(s.T(), dedicatedLineID, dedicatedGatheringByID.Lines.OrEmpty()[0].ID)
	})

	s.Run("GetInvoiceType resolves invoice types across storage tables", func() {
		ctx := s.T().Context()
		standardInvoiceType, err := s.BillingAdapter.GetInvoiceType(ctx, standardInvoice.GetInvoiceID())
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.InvoiceTypeStandard, standardInvoiceType)

		legacyGatheringInvoiceType, err := s.BillingAdapter.GetInvoiceType(ctx, legacyGatheringInvoice.GetInvoiceID())
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.InvoiceTypeGathering, legacyGatheringInvoiceType)

		dedicatedGatheringInvoiceType, err := s.BillingAdapter.GetInvoiceType(ctx, billing.InvoiceID{
			Namespace: gatheringNamespace,
			ID:        dedicatedInvoiceID,
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.InvoiceTypeGathering, dedicatedGatheringInvoiceType)
	})

	s.Run("ListGatheringInvoices hydrates legacy and dedicated gathering invoices", func() {
		listedGatheringInvoices, err := s.BillingAdapter.ListGatheringInvoices(s.T().Context(), billing.ListGatheringInvoicesInput{
			Namespaces: []string{gatheringNamespace},
			IDs:        []string{standardInvoice.ID, legacyGatheringInvoice.ID, dedicatedInvoiceID},
			Expand:     billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), 2, listedGatheringInvoices.TotalCount)
		require.Len(s.T(), listedGatheringInvoices.Items, 2)
		listedGatheringInvoicesByID := lo.SliceToMap(listedGatheringInvoices.Items, func(invoice billing.GatheringInvoice) (string, billing.GatheringInvoice) {
			return invoice.ID, invoice
		})
		require.Equal(s.T(), legacyGatheringInvoice.ID, listedGatheringInvoicesByID[legacyGatheringInvoice.ID].ID)
		require.Equal(s.T(), dedicatedInvoiceID, listedGatheringInvoicesByID[dedicatedInvoiceID].ID)
		require.Len(s.T(), listedGatheringInvoicesByID[dedicatedInvoiceID].Lines.OrEmpty(), 1)
		require.Equal(s.T(), dedicatedLineID, listedGatheringInvoicesByID[dedicatedInvoiceID].Lines.OrEmpty()[0].ID)
	})

	s.Run("GetGatheringLinesForSubscription loads dedicated gathering lines", func() {
		lines, err := s.BillingAdapter.GetGatheringLinesForSubscription(s.T().Context(), billing.GetLinesForSubscriptionInput{
			Namespace:      gatheringNamespace,
			SubscriptionID: subscriptionID,
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 1)
		require.Equal(s.T(), dedicatedLineID, lines[0].ID)
		require.Equal(s.T(), servicePeriodStart, lines[0].ServicePeriod.From)
		require.Equal(s.T(), servicePeriodEnd, lines[0].ServicePeriod.To)
		require.Equal(s.T(), &billing.SubscriptionReference{
			SubscriptionID: subscriptionID,
			PhaseID:        subscriptionPhase.ID,
			ItemID:         subscriptionItem.ID,
		}, lines[0].Subscription)
	})

	s.Run("GetUnpinnedCustomerIDsWithPaidSubscription finds a dedicated gathering line", func() {
		customerIDs, err := s.BillingAdapter.GetUnpinnedCustomerIDsWithPaidSubscription(s.T().Context(), billing.GetUnpinnedCustomerIDsWithPaidSubscriptionInput{
			Namespace: gatheringNamespace,
		})
		require.NoError(s.T(), err)
		require.ElementsMatch(s.T(), []customer.CustomerID{
			legacyGatheringInvoice.GetCustomerID(),
		}, customerIDs)
	})

	s.Run("GetGatheringInvoiceById rejects a standard invoice", func() {
		_, err := s.BillingAdapter.GetGatheringInvoiceById(s.T().Context(), billing.GetGatheringInvoiceByIdInput{
			Invoice: standardInvoice.GetInvoiceID(),
		})
		require.ErrorAs(s.T(), err, &billing.ValidationError{})
	})

	s.Run("GetGatheringInvoiceById returns not found for an unknown invoice", func() {
		_, err := s.BillingAdapter.GetGatheringInvoiceById(s.T().Context(), billing.GetGatheringInvoiceByIdInput{
			Invoice: billing.InvoiceID{
				Namespace: gatheringNamespace,
				ID:        ulid.Make().String(),
			},
		})
		require.ErrorAs(s.T(), err, &billing.NotFoundError{})
	})

	s.Run("ListStandardInvoices excludes gathering invoices from both storage tables", func() {
		standardOnly, err := s.BillingAdapter.ListStandardInvoices(s.T().Context(), billing.ListStandardInvoicesInput{
			Namespaces: []string{standardNamespace, gatheringNamespace},
			IDs:        []string{standardInvoice.ID, legacyGatheringInvoice.ID, dedicatedInvoiceID},
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), 1, standardOnly.TotalCount)
		require.Len(s.T(), standardOnly.Items, 1)
		require.Equal(s.T(), standardInvoice.ID, standardOnly.Items[0].ID)
	})

	s.Run("GetGatheringInvoiceById prefers a dedicated replacement over its legacy row", func() {
		ctx := s.T().Context()
		_, err := s.DBClient.BillingGatheringInvoice.Create().
			SetID(legacyGatheringInvoice.ID).
			SetNamespace(gatheringNamespace).
			SetName("Dedicated replacement gathering invoice").
			SetNumber("GATHERING-REPLACEMENT").
			SetCustomerID(legacyGatheringInvoice.GetCustomerID().ID).
			SetCurrency(currencyx.Code(currency.EUR)).
			SetSchemaLevel(2).
			Save(ctx)
		require.NoError(s.T(), err)

		replacedGatheringInvoice, err := s.BillingAdapter.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: legacyGatheringInvoice.GetInvoiceID(),
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), "GATHERING-REPLACEMENT", replacedGatheringInvoice.Number)
	})
}

type newLineInput struct {
	Namespace              string
	Period                 timeutil.ClosedPeriod
	Invoice                *billing.StandardInvoice
	Name                   string
	ChildUniqueReferenceID string
	DetailedLines          mo.Option[[]newLineInput]
}

func newLine(in newLineInput) *billing.StandardLine {
	out := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: in.Namespace,
				Name:      in.Name,
			}),

			ManagedBy: billing.ManuallyManagedLine,
			Engine:    billing.LineEngineTypeInvoice,
			InvoiceID: in.Invoice.ID,
			Currency:  in.Invoice.Currency,

			Period:    in.Period,
			InvoiceAt: in.Period.To,

			ChildUniqueReferenceID: lo.EmptyableToPtr(in.ChildUniqueReferenceID),
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(1),
			}),
			FeatureKey: "test",
		},
	}

	if in.DetailedLines.IsPresent() {
		// Make the line present, but empty (so that it's present even if DetailedLines is only present)
		out.DetailedLines = lo.Map(in.DetailedLines.OrEmpty(), func(d newLineInput, _ int) billing.DetailedLine {
			return newDetailedLine(d)
		})
	}

	return out
}

func newDetailedLine(in newLineInput) billing.DetailedLine {
	return billing.DetailedLine{
		DetailedLineBase: billing.DetailedLineBase{
			InvoiceID: in.Invoice.ID,
			Base: stddetailedline.Base{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: in.Namespace,
					Name:      in.Name,
				}),
				Currency:               in.Invoice.Currency,
				ServicePeriod:          in.Period,
				ChildUniqueReferenceID: in.ChildUniqueReferenceID,
				PerUnitAmount:          alpacadecimal.NewFromFloat(100),
				Quantity:               alpacadecimal.NewFromFloat(1),
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				Category:               stddetailedline.CategoryRegular,
			},
		},
	}
}

func (s *BillingAdapterTestSuite) TestDetailedLineHandling() {
	ctx := context.Background()
	ns := "ns-adapter-detailed-line"
	// Given we have an invoice

	period := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2023-01-10T00:00:00Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2023-01-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	// When we create a line with detailed fields those get persisted
	linesIn := []*billing.StandardLine{
		newLine(newLineInput{
			Namespace: ns,
			Period:    period,
			Invoice:   invoice,
			Name:      "Test Line 1",
			DetailedLines: mo.Some([]newLineInput{
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 1.1",
					ChildUniqueReferenceID: "ref1",
				},
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 1.2",
					ChildUniqueReferenceID: "ref2",
				},
			}),
		}),
		newLine(newLineInput{
			Namespace: ns,
			Period:    period,
			Invoice:   invoice,
			Name:      "Test Line 2",
			DetailedLines: mo.Some([]newLineInput{
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 2.1",
					ChildUniqueReferenceID: "ref1",
				},
			}),
		}),
		newLine(newLineInput{
			Namespace: ns,
			Period:    period,
			Invoice:   invoice,
			Name:      "Test Line 3",
			DetailedLines: mo.Some([]newLineInput{
				{
					Namespace:              ns,
					Period:                 period,
					Invoice:                invoice,
					Name:                   "Test Line 3.1",
					ChildUniqueReferenceID: "ref1",
				},
			}),
		}),
	}

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace:   ns,
		SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
		Lines:       linesIn,
		InvoiceID:   invoice.ID,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 3)

	// Then the lines are persisted as expected
	// Line 1
	require.Equal(s.T(), linesIn[0].Name, lines[0].Name)
	require.Empty(s.T(), linesIn[0].DetailedLines[0].ID)
	require.Len(s.T(), lines[0].DetailedLines, 2)
	require.ElementsMatch(s.T(),
		getUniqReferenceNames(linesIn[0].DetailedLines),
		getUniqReferenceNames(lines[0].DetailedLines))
	require.ElementsMatch(s.T(),
		getLineNames(linesIn[0].DetailedLines),
		getLineNames(lines[0].DetailedLines))

	require.Equal(s.T(), linesIn[1].Name, lines[1].Name)
	require.Len(s.T(), lines[1].DetailedLines, 1)
	require.ElementsMatch(s.T(),
		getUniqReferenceNames(linesIn[1].DetailedLines),
		getUniqReferenceNames(lines[1].DetailedLines))
	require.ElementsMatch(s.T(),
		getLineNames(linesIn[1].DetailedLines),
		getLineNames(lines[1].DetailedLines))

	require.Equal(s.T(), linesIn[2].Name, lines[2].Name)
	require.Len(s.T(), lines[2].DetailedLines, 1)

	// When we execute an upsert the detailed lines are updated, but not duplicated
	s.Run("Detailed line upserting", func() {
		unchangedDetailedLineUpdatedAt := lo.FindOrElse[billing.DetailedLine](lines[0].DetailedLines,
			billing.DetailedLine{},
			func(l billing.DetailedLine) bool {
				return l.ChildUniqueReferenceID == "ref1"
			},
		).UpdatedAt.Unix()

		newDetailedLine := newDetailedLine(newLineInput{
			Namespace:              ns,
			Period:                 period,
			Invoice:                invoice,
			Name:                   "Test Line 1.3",
			ChildUniqueReferenceID: "ref3",
		})

		lineChildren := lines[0].DetailedLines
		lineChildren = append(lineChildren, newDetailedLine)

		lines[0].DetailedLines = lines[0].DetailedLinesWithIDReuse(lineChildren)

		// Set to empty array => detailed lines should be deleted
		lines[1].DetailedLines = nil
		lines[2].DetailedLines = billing.DetailedLines{}

		// When we persist the changes
		lines, err = s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace:   ns,
			Lines:       lines,
			SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
			InvoiceID:   invoice.ID,
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 3)

		// Then the lines are persisted as expected
		require.Len(s.T(), lines[0].DetailedLines, 3)
		require.ElementsMatch(s.T(),
			getUniqReferenceNames(lineChildren),
			getUniqReferenceNames(lines[0].DetailedLines))

		require.Equal(s.T(), lo.CountBy(lines[0].DetailedLines, func(l billing.DetailedLine) bool {
			return l.ID != ""
		}), 3, "all lines must have IDs set")

		// Then ref1 has not been changed
		require.Equal(s.T(), unchangedDetailedLineUpdatedAt, lo.FindOrElse(lines[0].DetailedLines,
			billing.DetailedLine{},
			func(l billing.DetailedLine) bool {
				return l.ChildUniqueReferenceID == "ref1"
			}).UpdatedAt.Unix())

		require.Len(s.T(), lines[1].DetailedLines, 0)
		require.Len(s.T(), lines[2].DetailedLines, 0)
	})

	// When we remove a detailed line, the line gets deleted
	s.Run("Detailed line update (still a removal case)", func() {
		detailedLines := lines[0].DetailedLines

		slices.SortFunc(detailedLines, func(a, b billing.DetailedLine) int {
			return strings.Compare(a.ChildUniqueReferenceID, b.ChildUniqueReferenceID)
		})

		require.Len(s.T(), detailedLines, 3)
		require.Equal(s.T(), "ref1", detailedLines[0].ChildUniqueReferenceID)

		// Replace the first detailed line with a new child
		detailedLines[0] = newDetailedLine(newLineInput{
			Namespace:              ns,
			Period:                 period,
			Invoice:                invoice,
			Name:                   "Test Line 1.4",
			ChildUniqueReferenceID: "ref4",
		})

		childrenWithIDReuse := lines[0].DetailedLinesWithIDReuse(detailedLines)
		lines[0].DetailedLines = childrenWithIDReuse

		// When we persist the changes
		lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace:   ns,
			SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
			Lines:       []*billing.StandardLine{lines[0]},
			InvoiceID:   invoice.ID,
		})

		// Then we only get three lines
		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 1)
		require.Len(s.T(), lines[0].DetailedLines, 3)

		require.ElementsMatch(s.T(),
			getUniqReferenceNames(detailedLines),
			getUniqReferenceNames(lines[0].DetailedLines))
		require.ElementsMatch(s.T(),
			getLineNames(detailedLines),
			getLineNames(lines[0].DetailedLines))

		// When we query the line's children, we get the 4 lines, one is deleted
		lines, err = s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace:      ns,
			LineIDs:        []string{lines[0].ID},
			IncludeDeleted: true,
		})

		require.NoError(s.T(), err)
		require.Len(s.T(), lines, 1)
		childLines := lines[0].DetailedLines

		// Then we get the 4 lines
		require.Len(s.T(), childLines, 3)
		require.ElementsMatch(s.T(),
			[]string{"ref2", "ref3", "ref4"},
			getUniqReferenceNames(childLines))
	})
}

func (s *BillingAdapterTestSuite) TestUnitConfigSnapshotHandling() {
	ctx := s.T().Context()
	ns := "ns-adapter-unit-config"

	period := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2023-01-10T00:00:00Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2023-01-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	lineWithConfig := newLine(newLineInput{
		Namespace: ns,
		Period:    period,
		Invoice:   invoice,
		Name:      "Line with unit config",
	})
	lineWithConfig.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7400))
	lineWithConfig.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(8))
	lineWithConfig.UsageBased.UnitConfig = unitConfig

	lineWithoutConfig := newLine(newLineInput{
		Namespace: ns,
		Period:    period,
		Invoice:   invoice,
		Name:      "Line without unit config",
	})

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace:   ns,
		SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
		Lines:       []*billing.StandardLine{lineWithConfig, lineWithoutConfig},
		InvoiceID:   invoice.ID,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 2)

	s.Run("snapshot round-trips through persistence", func() {
		readBack, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace: ns,
			LineIDs:   []string{lines[0].ID},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), readBack, 1)

		require.NotNil(s.T(), readBack[0].UsageBased.UnitConfig)
		require.True(s.T(), unitConfig.Equal(readBack[0].UsageBased.UnitConfig))

		// GetUnitConfig is the rating-path accessor: re-rating must convert from raw
		// through the persisted snapshot.
		require.True(s.T(), unitConfig.Equal(readBack[0].GetUnitConfig()))
	})

	s.Run("line without a unit config stays nil", func() {
		readBack, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace: ns,
			LineIDs:   []string{lines[1].ID},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), readBack, 1)

		require.Nil(s.T(), readBack[0].UsageBased.UnitConfig)
		require.Nil(s.T(), readBack[0].GetUnitConfig())
	})

	s.Run("re-upserting with a changed config updates the snapshot", func() {
		// unit_config is mutable on a draft line like price: a re-upsert carrying a
		// different config replaces the stored snapshot (invoice edits / charges patching).
		changedConfig := &productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationDivide,
			ConversionFactor: alpacadecimal.NewFromInt(500),
			Rounding:         productcatalog.UnitConfigRoundingModeFloor,
		}

		updated := lo.Must(lines[0].Clone())
		updated.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(9400))
		updated.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(10))
		updated.UsageBased.UnitConfig = changedConfig

		_, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace:   ns,
			SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
			Lines:       []*billing.StandardLine{updated},
			InvoiceID:   invoice.ID,
		})
		require.NoError(s.T(), err)

		readBack, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace: ns,
			LineIDs:   []string{lines[0].ID},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), readBack, 1)

		require.Equal(s.T(), float64(9400), lo.FromPtr(readBack[0].UsageBased.MeteredQuantity).InexactFloat64())
		require.NotNil(s.T(), readBack[0].UsageBased.UnitConfig)
		require.True(s.T(), changedConfig.Equal(readBack[0].UsageBased.UnitConfig))
	})

	s.Run("a mixed nil+config batch resolves each line independently", func() {
		// Finding-#1 guard: UpdateUnitConfig makes the conflict clause resolve this column
		// per row from that row's own excluded value, so batch composition cannot
		// cross-contaminate. Dropping the config on lines[0] and setting one on lines[1] in
		// ONE batch must clear lines[0] and set lines[1] — never leak lines[1]'s config onto
		// lines[0] via the shared insert column union.
		clearLine := lo.Must(lines[0].Clone())
		clearLine.UsageBased.UnitConfig = nil

		setLine := lo.Must(lines[1].Clone())
		setLine.UsageBased.UnitConfig = unitConfig

		_, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace:   ns,
			SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
			Lines:       []*billing.StandardLine{clearLine, setLine},
			InvoiceID:   invoice.ID,
		})
		require.NoError(s.T(), err)

		readBack, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace: ns,
			LineIDs:   []string{lines[0].ID, lines[1].ID},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), readBack, 2)

		byID := lo.KeyBy(readBack, func(l *billing.StandardLine) string { return l.ID })
		require.Nil(s.T(), byID[lines[0].ID].UsageBased.UnitConfig)
		require.NotNil(s.T(), byID[lines[1].ID].UsageBased.UnitConfig)
		require.True(s.T(), unitConfig.Equal(byID[lines[1].ID].UsageBased.UnitConfig))
	})
}

// TestStandardLineOptionalFieldMutability checks that optional mutable fields on the standard
// invoice-line upsert (here: description) are re-resolved on conflict. The clear case is the one
// that requires the explicit UpdateDescription() — a nil field is absent from the batch's insert
// union, so bare ResolveWithNewValues would leave the stale value.
func (s *BillingAdapterTestSuite) TestStandardLineOptionalFieldMutability() {
	ctx := s.T().Context()
	ns := "ns-adapter-line-optional-fields"

	period := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2023-02-10T00:00:00Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2023-02-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	line := newLine(newLineInput{
		Namespace: ns,
		Period:    period,
		Invoice:   invoice,
		Name:      "Line with description",
	})
	line.Description = lo.ToPtr("original")

	created, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace:   ns,
		SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
		Lines:       []*billing.StandardLine{line},
		InvoiceID:   invoice.ID,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), created, 1)

	readDescription := func() *string {
		readBack, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
			Namespace: ns,
			LineIDs:   []string{created[0].ID},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), readBack, 1)
		return readBack[0].Description
	}

	require.Equal(s.T(), "original", lo.FromPtr(readDescription()))

	s.Run("a changed description is updated on re-upsert", func() {
		updated := lo.Must(created[0].Clone())
		updated.Description = lo.ToPtr("updated")

		_, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace:   ns,
			SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
			Lines:       []*billing.StandardLine{updated},
			InvoiceID:   invoice.ID,
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), "updated", lo.FromPtr(readDescription()))
	})

	s.Run("a removed description is cleared on re-upsert", func() {
		updated := lo.Must(created[0].Clone())
		updated.Description = nil

		_, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
			Namespace:   ns,
			SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
			Lines:       []*billing.StandardLine{updated},
			InvoiceID:   invoice.ID,
		})
		require.NoError(s.T(), err)
		require.Nil(s.T(), readDescription())
	})
}

func getUniqReferenceNames(lines []billing.DetailedLine) []string {
	return lo.Map(lines, func(l billing.DetailedLine, _ int) string {
		return l.ChildUniqueReferenceID
	})
}

func getLineNames(lines []billing.DetailedLine) []string {
	return lo.Map(lines, func(l billing.DetailedLine, _ int) string {
		return l.Name
	})
}

// TestDiscountHandling tests the handling of discounts in the billing adapter
// this specific scenario simulates the detailed line calculation specifically (that's why we are
// using the detailed line's discounts for testing)
func (s *BillingAdapterTestSuite) TestDiscountHandling() {
	ctx := context.Background()
	ns := "ns-adapter-discount-handling"
	// Given we have an invoice

	period := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2023-01-10T00:00:00Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2023-01-20T00:00:00Z")),
	}

	invoice := s.setupInvoice(ctx, ns)

	// When we create a line with detailed fields those get persisted
	lineIn := newLine(newLineInput{
		Namespace: ns,
		Period:    period,
		Invoice:   invoice,
		Name:      "Test Line 1",

		DetailedLines: mo.Some([]newLineInput{
			{
				Namespace: ns,
				Period:    period,
				Invoice:   invoice,
				Name:      "Test Line 1.1",
				// Warning: this is required, as otherwise the line would be considered a new line
				ChildUniqueReferenceID: "ref1",
			},
		}),
	})

	manualDiscountName := "Test Discount 3 - manual"

	lineIn.DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{
		{
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(10),
				LineDiscountBase: billing.LineDiscountBase{
					Description:            lo.ToPtr("Test Discount 1"),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
				},
			},
		},
		{
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(20),
				LineDiscountBase: billing.LineDiscountBase{
					Description:            lo.ToPtr("Test Discount 2"),
					ChildUniqueReferenceID: lo.ToPtr("max-spend-multiline"),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
				},
			},
		},
		{
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(30),
				LineDiscountBase: billing.LineDiscountBase{
					Description: lo.ToPtr(manualDiscountName),
					Reason: billing.NewDiscountReasonFrom(productcatalog.PercentageDiscount{
						Percentage: models.NewPercentage(10),
					}),
				},
			},
		},
	}

	lines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace:   ns,
		SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
		Lines:       []*billing.StandardLine{lineIn},
		InvoiceID:   invoice.ID,
	})

	// Then the lines are persisted as expected
	require.NoError(s.T(), err)
	require.Len(s.T(), lines, 1)

	// Then the discounts are persisted as expected
	persistedDiscounts := lines[0].DetailedLines[0].AmountDiscounts
	require.Len(s.T(), persistedDiscounts, 3)

	// Remove the managed fields
	discountContents, err := persistedDiscounts.Mutate(func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
		discount.ManagedModelWithID = models.ManagedModelWithID{}
		return discount, nil
	})
	require.NoError(s.T(), err)

	inputDiscountContents := lo.Must(lineIn.DetailedLines[0].AmountDiscounts.Mutate(
		func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
			discount.ManagedModelWithID = models.ManagedModelWithID{}
			return discount, nil
		},
	))

	require.ElementsMatch(s.T(),
		discountContents,
		inputDiscountContents,
	)

	// Let's update the discounts
	childLine := lines[0].DetailedLines[0].Clone()

	// Let's find the existing manual discount's ID
	existingDiscountID := ""
	for _, discount := range persistedDiscounts {
		if discount.Description != nil && *discount.Description == manualDiscountName {
			existingDiscountID = discount.ID
			break
		}
	}
	require.NotEmpty(s.T(), existingDiscountID)

	childLine.AmountDiscounts = billing.AmountLineDiscountsManaged{
		{
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(30),
				LineDiscountBase: billing.LineDiscountBase{
					Description:            lo.ToPtr("Test Discount 1 v2"),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
				},
			},
		},
		// Maximum spend is deleted
		{
			ManagedModelWithID: models.ManagedModelWithID{
				ID: existingDiscountID,
			},
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(40),
				LineDiscountBase: billing.LineDiscountBase{
					Description: lo.ToPtr("Test Discount 3 - updated"),
					Reason: billing.NewDiscountReasonFrom(productcatalog.PercentageDiscount{
						Percentage: models.NewPercentage(10),
					}),
				},
			},
		},
		{
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: alpacadecimal.NewFromFloat(50),
				LineDiscountBase: billing.LineDiscountBase{
					Description: lo.ToPtr("Test Discount 4 - manual"),
					Reason: billing.NewDiscountReasonFrom(productcatalog.PercentageDiscount{
						Percentage: models.NewPercentage(20),
					}),
				},
			},
		},
	}

	updateLineIn := lo.Must(lines[0].Clone())
	childrenWithIDReuse := updateLineIn.DetailedLinesWithIDReuse(
		billing.DetailedLines{childLine},
	)
	updateLineIn.DetailedLines = childrenWithIDReuse

	updatedLines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace:   ns,
		SchemaLevel: billingadapter.DefaultInvoiceWriteSchemaLevel,
		Lines:       []*billing.StandardLine{updateLineIn},
		InvoiceID:   invoice.ID,
	})

	// Then the discounts are persisted as expected
	require.NoError(s.T(), err)
	require.Len(s.T(), updatedLines, 1)

	previousVersionDiscounts := persistedDiscounts
	persistedDiscounts = updatedLines[0].DetailedLines[0].AmountDiscounts
	require.Len(s.T(), persistedDiscounts, 3)

	expectedChildLineDiscounts := childLine.AmountDiscounts
	// Line 0: we expect that the ID is set to the same value
	previousVersion, ok := previousVersionDiscounts.GetDiscountByChildUniqueReferenceID(billing.LineMaximumSpendReferenceID)
	require.True(s.T(), ok)

	currentVersion := s.findAmountDiscountByDescription(persistedDiscounts, "Test Discount 1 v2")
	require.True(s.T(), expectedChildLineDiscounts[0].ContentsEqual(currentVersion))
	require.Equal(s.T(), currentVersion.ManagedModelWithID, models.ManagedModelWithID{
		ID: previousVersion.GetID(),
		ManagedModel: models.ManagedModel{
			// CreatedAt is unchanged
			CreatedAt: previousVersion.CreatedAt,
			UpdatedAt: currentVersion.UpdatedAt,
		},
	})

	// Line 1: maximum spend with retained id
	previousVersion = s.findAmountDiscountByDescription(previousVersionDiscounts, "Test Discount 3 - manual")
	currentVersion = s.findAmountDiscountByDescription(persistedDiscounts, "Test Discount 3 - updated")
	require.True(s.T(), expectedChildLineDiscounts[1].ContentsEqual(currentVersion))
	require.Equal(s.T(), currentVersion.ManagedModelWithID, models.ManagedModelWithID{
		ID: previousVersion.GetID(),
		ManagedModel: models.ManagedModel{
			// CreatedAt is unchanged
			CreatedAt: previousVersion.CreatedAt,
			UpdatedAt: currentVersion.UpdatedAt,
		},
	})

	// Line 2: new discount
	currentVersion = s.findAmountDiscountByDescription(persistedDiscounts, "Test Discount 4 - manual")
	require.True(s.T(), expectedChildLineDiscounts[2].ContentsEqual(currentVersion))

	require.ElementsMatch(s.T(),
		lo.Must(
			expectedChildLineDiscounts.Mutate(
				func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
					discount.ManagedModelWithID = models.ManagedModelWithID{}
					return discount, nil
				},
			),
		),
		lo.Must(
			persistedDiscounts.Mutate(
				func(discount billing.AmountLineDiscountManaged) (billing.AmountLineDiscountManaged, error) {
					discount.ManagedModelWithID = models.ManagedModelWithID{}
					return discount, nil
				},
			),
		),
	)
}

func (s *BillingAdapterTestSuite) findAmountDiscountByDescription(discounts []billing.AmountLineDiscountManaged, description string) billing.AmountLineDiscountManaged {
	s.T().Helper()

	for _, discount := range discounts {
		if discount.Description != nil && *discount.Description == description {
			return discount
		}
	}

	s.T().Fatalf("discount not found: %s", description)
	return billing.AmountLineDiscountManaged{}
}

func (s *BillingAdapterTestSuite) TestHardDeleteGatheringInvoiceLines() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-adapter-hard-delete-gathering-invoice-lines")
	featureKey := "in-advance-payment"

	var customerEntity *customer.Customer

	s.Run("Given a customer and default billing profile exists", func() {
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

		customerEntity = s.CreateTestCustomer(namespace, "test-customer")
		s.NotNil(customerEntity)

		_ = lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      featureKey,
			Key:       featureKey,
		}))
	})

	var gatheringInvoice billing.GatheringInvoice
	s.Run("Given a gathering invoice with two lines", func() {
		periodStart := time.Now().Add(-time.Hour)
		periodEnd := time.Now().Add(time.Hour)

		res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test line 1",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     periodEnd,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
						FeatureKey: featureKey,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						})),
					},
				},
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test line 2",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     periodEnd,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
						FeatureKey: featureKey,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						})),
					},
				},
			},
		})
		s.NoError(err)

		gatheringInvoice = res.Invoice
	})

	var (
		deletedLine                     billing.GatheringLine
		gatheringInvoiceWithDeletedLine billing.GatheringInvoice
	)
	s.Run("When we hard delete one of the lines", func() {
		deletedLine = gatheringInvoice.Lines.OrEmpty()[0]
		err := s.BillingAdapter.HardDeleteGatheringInvoiceLines(ctx, gatheringInvoice.GetInvoiceID(), []string{deletedLine.ID})
		s.NoError(err)

		gatheringInvoice, err = s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoice.GetInvoiceID(),
			Expand:  billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		gatheringInvoiceWithDeletedLine = gatheringInvoice
	})

	s.Run("Then the gathering invoice has only one line", func() {
		s.Len(gatheringInvoiceWithDeletedLine.Lines.OrEmpty(), 1)
		s.NotEqual(deletedLine.ID, gatheringInvoice.Lines.OrEmpty()[0].ID)
	})

	s.Run("Then the deleted line's usage based config is also deleted", func() {
		s.NotEmpty(deletedLine.UBPConfigID)

		_, err := s.DBClient.BillingInvoiceUsageBasedLineConfig.Query().
			Where(billinginvoiceusagebasedlineconfig.ID(deletedLine.UBPConfigID)).
			Only(ctx)

		s.Error(err)
		s.True(db.IsNotFound(err))
	})
}

func (s *BillingAdapterTestSuite) TestHardDeleteGatheringInvoiceLinesNegative() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("ns-adapter-hard-delete-gathering-invoice-lines-negative")
	featureKey := "in-advance-payment"

	now := lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z"))
	clock.SetTime(now)
	defer clock.ResetTime()

	var customerEntity *customer.Customer

	s.Run("Given a customer and  billing profile exists", func() {
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

		customerEntity = s.CreateTestCustomer(namespace, "test-customer")
		s.NotNil(customerEntity)

		_ = lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      featureKey,
			Key:       featureKey,
		}))
	})

	var (
		gatheringInvoice billing.GatheringInvoice
		standardInvoice  billing.StandardInvoice
	)

	s.Run("Given a gathering invoice with two lines", func() {
		line1PeriodStart := lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z"))
		line1PeriodEnd := line1PeriodStart.Add(time.Hour)

		line2PeriodStart := lo.Must(time.Parse(time.RFC3339, "2026-01-01T01:00:00Z"))
		line2PeriodEnd := line2PeriodStart.Add(time.Hour)

		createdPendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test line 1",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: line1PeriodStart, To: line1PeriodEnd},
						InvoiceAt:     line1PeriodStart,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
						FeatureKey: featureKey,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(100),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						})),
					},
				},
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Test line 2",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: line2PeriodStart, To: line2PeriodEnd},
						InvoiceAt:     line2PeriodStart,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						RateCardDiscounts: billing.Discounts{
							Percentage: &billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
							},
						},
						FeatureKey: featureKey,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(100),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						})),
					},
				},
			},
		})
		s.NoError(err)

		standardInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
		})
		s.NoError(err)
		s.Len(standardInvoices, 1)

		standardInvoice = standardInvoices[0]

		gatheringInvoice, err = s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: createdPendingLines.Invoice.GetInvoiceID(),
			Expand:  billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.NotNil(gatheringInvoice)
		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
	})

	s.Run("When we try to hard delete a line from the standard invoice then we fail", func() {
		err := s.BillingAdapter.HardDeleteGatheringInvoiceLines(ctx, standardInvoice.GetInvoiceID(), []string{standardInvoice.Lines.OrEmpty()[0].ID})
		s.Error(err)
	})

	s.Run("When we try to delete a line that does not belong to the invoice then we fail", func() {
		err := s.BillingAdapter.HardDeleteGatheringInvoiceLines(ctx, gatheringInvoice.GetInvoiceID(), []string{standardInvoice.Lines.OrEmpty()[0].ID})
		s.Error(err)
	})
}
