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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	ombilling "github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type LineEngineTestSuite struct {
	BaseSuite
}

func TestLineEngine(t *testing.T) {
	suite.Run(t, new(LineEngineTestSuite))
}

type mockCollectionCompletedLineEngine struct {
	mock.Mock
	engineType ombilling.LineEngineType
}

func mustAsNewStandardLines(input ombilling.BuildStandardInvoiceLinesInput) ombilling.StandardLines {
	out := make(ombilling.StandardLines, 0, len(input.GatheringLines))
	for _, gatheringLine := range input.GatheringLines {
		stdLine, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
		if err != nil {
			panic(err)
		}

		out = append(out, stdLine)
	}

	return out
}

func (m *mockCollectionCompletedLineEngine) GetLineEngineType() ombilling.LineEngineType {
	if m.engineType == "" {
		panic("engine type is required")
	}

	return m.engineType
}

func (m *mockCollectionCompletedLineEngine) IsLineBillableAsOf(_ context.Context, input ombilling.IsLineBillableAsOfInput) (bool, error) {
	return !lo.IsEmpty(input.ResolvedBillablePeriod), nil
}

func (m *mockCollectionCompletedLineEngine) SplitGatheringLine(_ context.Context, _ ombilling.SplitGatheringLineInput) (ombilling.SplitGatheringLineResult, error) {
	return ombilling.SplitGatheringLineResult{}, fmt.Errorf("split is not supported")
}

func (m *mockCollectionCompletedLineEngine) BuildStandardInvoiceLines(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
	args := m.Called(input.Invoice, input.GatheringLines)
	lines := args.Get(0)

	return lines.(ombilling.StandardLines), args.Error(1)
}

func (m *mockCollectionCompletedLineEngine) OnCollectionCompleted(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
	args := m.Called(input.Invoice, input.Lines)
	lines := args.Get(0)

	return lines.(ombilling.StandardLines), args.Error(1)
}

func (m *mockCollectionCompletedLineEngine) CalculateLines(input ombilling.CalculateLinesInput) (ombilling.StandardLines, error) {
	return input.Lines, nil
}

func (s *LineEngineTestSuite) registerMockLineEngine(t *testing.T, engine ombilling.LineEngine) {
	t.Helper()
	s.Require().NoError(s.BillingService.RegisterLineEngine(engine))
}

func (s *LineEngineTestSuite) unregisterLineEngine(t *testing.T, engine ombilling.LineEngine) {
	t.Helper()
	s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
}

func (s *LineEngineTestSuite) createMeteredDraftInvoiceWaitingForCollection(
	ctx context.Context,
	namespace string,
	engineType ombilling.LineEngineType,
	lineName string,
) (ombilling.StandardInvoice, time.Time) {
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	meterSlug := fmt.Sprintf("%s-meter", namespace)
	meterID := ulid.Make().String()
	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: meterID,
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Line Engine Test Meter",
		},
		Key:           meterSlug,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	}})
	s.Require().NoError(err)

	testFeature := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s-feature", namespace),
		Key:       fmt.Sprintf("%s-feature", namespace),
		MeterID:   lo.ToPtr(meterID),
	}))

	customerEntity := s.CreateTestCustomer(namespace, "test-subject-1")

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithCollectionInterval(datetime.NewISODuration(0, 0, 0, 1, 0, 0, 0)))

	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T11:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-02T13:13:14Z"))

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 10, periodStart.Add(-time.Minute))

	pendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx, ombilling.CreatePendingInvoiceLinesInput{
		Customer: customer.CustomerID{
			Namespace: customerEntity.Namespace,
			ID:        customerEntity.ID,
		},
		Currency: currencyx.Code(currency.USD),
		Lines: []ombilling.GatheringLine{{
			GatheringLineBase: ombilling.GatheringLineBase{
				ManagedResource: models.ManagedResource{
					NamespacedModel: models.NamespacedModel{Namespace: namespace},
					Name:            lineName,
				},
				ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
				InvoiceAt:     periodEnd,
				ManagedBy:     ombilling.ManuallyManagedLine,
				FeatureKey:    testFeature.Key,
				Engine:        engineType,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				})),
			},
		}},
	})
	s.Require().NoError(err)
	s.Require().Len(pendingLines.Lines, 1)

	clock.SetTime(periodEnd)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, ombilling.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Equal(ombilling.StandardInvoiceStatusDraftWaitingForCollection, invoices[0].Status)

	return invoices[0], invoices[0].DefaultCollectionAtForStandardInvoice()
}

func (s *LineEngineTestSuite) TestCollectionCompletedErrorsBecomeValidationIssues() {
	var (
		ctx          = context.Background()
		namespace    = s.GetUniqueNamespace("ns-line-engine-collection-completed-validation")
		mockEngine   = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeUsageBased}
		invoice      ombilling.StandardInvoice
		collectionAt time.Time
		err          error
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	defer mockEngine.AssertExpectations(s.T())
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a failing collection-completed engine", func() {
		mockEngine.On("BuildStandardInvoiceLines", mock.Anything, mock.Anything).Return(
			func(invoice ombilling.StandardInvoice, gatheringLines ombilling.GatheringLines) ombilling.StandardLines {
				return mustAsNewStandardLines(ombilling.BuildStandardInvoiceLinesInput{
					Invoice:        invoice,
					GatheringLines: gatheringLines,
				})
			},
			nil,
		).Once()

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - mock collection completed",
		)
	})

	s.Run("When collection is completed", func() {
		mockEngine.On("OnCollectionCompleted", mock.Anything, mock.Anything).Return(ombilling.StandardLines(nil), fmt.Errorf("mock collection completed failure")).Once()

		clock.SetTime(collectionAt.Add(time.Minute))
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
	})

	s.Run("Then the engine failure becomes a validation issue", func() {
		s.Equal(ombilling.StandardInvoiceStatusDraftInvalid, invoice.Status)
		s.Len(invoice.ValidationIssues, 1)
		s.Equal("mock collection completed failure", invoice.ValidationIssues[0].Message)
		s.Equal(ombilling.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
		s.Equal(ombilling.LineEngineValidationComponent(ombilling.LineEngineTypeChargeUsageBased), invoice.ValidationIssues[0].Component)
	})
}

func (s *LineEngineTestSuite) TestCollectionCompletedCustomSnapshotIsPreserved() {
	var (
		ctx          = context.Background()
		namespace    = s.GetUniqueNamespace("ns-line-engine-custom-snapshot-preserved")
		mockEngine   = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		invoice      ombilling.StandardInvoice
		collectionAt time.Time
		err          error
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	defer mockEngine.AssertExpectations(s.T())
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a custom collection-completed engine", func() {
		mockEngine.On("BuildStandardInvoiceLines", mock.Anything, mock.Anything).Return(
			func(invoice ombilling.StandardInvoice, gatheringLines ombilling.GatheringLines) ombilling.StandardLines {
				return mustAsNewStandardLines(ombilling.BuildStandardInvoiceLinesInput{
					Invoice:        invoice,
					GatheringLines: gatheringLines,
				})
			},
			nil,
		).Once()

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - custom collection snapshot",
		)
	})

	s.Run("When collection is completed", func() {
		mockEngine.On("OnCollectionCompleted", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			lines := args.Get(1).(ombilling.StandardLines)
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}
		}).Return(func(_ ombilling.StandardInvoice, lines ombilling.StandardLines) ombilling.StandardLines {
			return lines
		}, nil).Once()

		clock.FreezeTime(collectionAt.Add(time.Minute).UTC())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
	})

	s.Run("Then the returned snapshot is preserved on the invoice", func() {
		s.NotNil(invoice.QuantitySnapshotedAt)
		s.Equal(collectionAt.Add(time.Minute).UTC(), *invoice.QuantitySnapshotedAt)
		s.Len(invoice.Lines.OrEmpty(), 1)
		s.NotNil(invoice.Lines.OrEmpty()[0].UsageBased)
		s.Equal(alpacadecimal.NewFromInt(7), lo.FromPtr(invoice.Lines.OrEmpty()[0].UsageBased.Quantity))
		s.Equal(alpacadecimal.NewFromInt(7), lo.FromPtr(invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity))
	})
}
