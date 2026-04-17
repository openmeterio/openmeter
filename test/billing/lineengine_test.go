package billing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
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
	engineType ombilling.LineEngineType

	buildStandardInvoiceLines func(ctx context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error)
	onStandardInvoiceCreated  func(ctx context.Context, input ombilling.OnStandardInvoiceCreatedInput) (ombilling.StandardLines, error)
	onCollectionCompleted     func(ctx context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error)
	onInvoiceIssued           func(ctx context.Context, input ombilling.OnInvoiceIssuedInput) error
	onPaymentAuthorized       func(ctx context.Context, input ombilling.OnPaymentAuthorizedInput) error
	onPaymentSettled          func(ctx context.Context, input ombilling.OnPaymentSettledInput) error
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

func (m *mockCollectionCompletedLineEngine) BuildStandardInvoiceLines(ctx context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
	if m.buildStandardInvoiceLines == nil {
		return nil, errors.New("buildStandardInvoiceLines is not set")
	}

	return m.buildStandardInvoiceLines(ctx, input)
}

func (m *mockCollectionCompletedLineEngine) OnCollectionCompleted(ctx context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
	if m.onCollectionCompleted == nil {
		return nil, errors.New("onCollectionCompleted is not set")
	}

	return m.onCollectionCompleted(ctx, input)
}

func (m *mockCollectionCompletedLineEngine) OnStandardInvoiceCreated(ctx context.Context, input ombilling.OnStandardInvoiceCreatedInput) (ombilling.StandardLines, error) {
	if m.onStandardInvoiceCreated == nil {
		return input.Lines, nil
	}

	return m.onStandardInvoiceCreated(ctx, input)
}

func (m *mockCollectionCompletedLineEngine) OnInvoiceIssued(ctx context.Context, input ombilling.OnInvoiceIssuedInput) error {
	if m.onInvoiceIssued == nil {
		return fmt.Errorf("onInvoiceIssued is not set")
	}

	return m.onInvoiceIssued(ctx, input)
}

func (m *mockCollectionCompletedLineEngine) OnPaymentAuthorized(ctx context.Context, input ombilling.OnPaymentAuthorizedInput) error {
	if m.onPaymentAuthorized == nil {
		return nil
	}

	return m.onPaymentAuthorized(ctx, input)
}

func (m *mockCollectionCompletedLineEngine) OnPaymentSettled(ctx context.Context, input ombilling.OnPaymentSettledInput) error {
	if m.onPaymentSettled == nil {
		return nil
	}

	return m.onPaymentSettled(ctx, input)
}

func (m *mockCollectionCompletedLineEngine) CalculateLines(input ombilling.CalculateLinesInput) (ombilling.StandardLines, error) {
	return input.Lines, nil
}

func (m *mockCollectionCompletedLineEngine) Reset() {
	*m = mockCollectionCompletedLineEngine{
		engineType: m.engineType,
	}
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

	return s.createMeteredDraftInvoiceWaitingForCollectionForApp(ctx, namespace, sandboxApp.GetID(), engineType, lineName)
}

func (s *LineEngineTestSuite) createMeteredDraftInvoiceWaitingForCollectionForApp(
	ctx context.Context,
	namespace string,
	appID app.AppID,
	engineType ombilling.LineEngineType,
	lineName string,
) (ombilling.StandardInvoice, time.Time) {
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

	s.ProvisionBillingProfile(ctx, namespace, appID, WithCollectionInterval(datetime.NewISODuration(0, 0, 0, 1, 0, 0, 0)))

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

func (s *LineEngineTestSuite) markInvoicePaid(ctx context.Context, invoiceID ombilling.InvoiceID) ombilling.StandardInvoice {
	appType := s.mustGetInvoiceAppType(ctx, invoiceID)

	s.Require().NoError(s.BillingService.TriggerInvoice(ctx, ombilling.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: ombilling.InvoiceTriggerInput{
			Invoice: invoiceID,
			Trigger: ombilling.TriggerPaid,
		},
		AppType:    appType,
		Capability: app.CapabilityTypeCollectPayments,
	}))

	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, ombilling.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
	})
	s.Require().NoError(err)

	return invoice
}

func (s *LineEngineTestSuite) markInvoiceAuthorized(ctx context.Context, invoiceID ombilling.InvoiceID) ombilling.StandardInvoice {
	appType := s.mustGetInvoiceAppType(ctx, invoiceID)

	s.Require().NoError(s.BillingService.TriggerInvoice(ctx, ombilling.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: ombilling.InvoiceTriggerInput{
			Invoice: invoiceID,
			Trigger: ombilling.TriggerAuthorized,
		},
		AppType:    appType,
		Capability: app.CapabilityTypeCollectPayments,
	}))

	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, ombilling.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
	})
	s.Require().NoError(err)

	return invoice
}

func (s *LineEngineTestSuite) mustGetInvoiceAppType(ctx context.Context, invoiceID ombilling.InvoiceID) app.AppType {
	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, ombilling.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
	})
	s.Require().NoError(err)

	if invoice.Workflow.Apps != nil && invoice.Workflow.Apps.Invoicing != nil {
		return invoice.Workflow.Apps.Invoicing.GetType()
	}

	invoicingApp, err := s.AppService.GetApp(ctx, app.GetAppInput{
		Namespace: invoice.Namespace,
		ID:        invoice.Workflow.AppReferences.Invoicing.ID,
	})
	s.Require().NoError(err)

	return invoicingApp.GetType()
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
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a failing collection-completed engine", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(ombilling.BuildStandardInvoiceLinesInput{
				Invoice:        input.Invoice,
				GatheringLines: input.GatheringLines,
			}), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - mock collection completed",
		)
	})

	s.Run("When collection is completed", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			return nil, fmt.Errorf("mock collection completed failure")
		}

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
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a custom collection-completed engine", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(ombilling.BuildStandardInvoiceLinesInput{
				Invoice:        input.Invoice,
				GatheringLines: input.GatheringLines,
			}), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - custom collection snapshot",
		)
	})

	s.Run("When collection is completed", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

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

func (s *LineEngineTestSuite) TestOnInvoiceIssuedIsCalled() {
	var (
		ctx          = s.T().Context()
		namespace    = s.GetUniqueNamespace("ns-line-engine-on-invoice-issued")
		mockEngine   = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		invoice      ombilling.StandardInvoice
		collectionAt time.Time
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with an issued hook", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - invoice issued hook",
		)
	})

	s.Run("When the invoice is collected and then issued", func() {
		defer mockEngine.Reset()
		onInvoiceIssuedCnt := 0

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			onInvoiceIssuedCnt++
			s.Equal(input.Invoice.ID, invoice.ID)
			s.Len(input.Lines, 1)
			s.Equal(input.Invoice.ID, input.Lines[0].InvoiceID)
			s.Equal(mockEngine.GetLineEngineType(), input.Lines[0].Engine)
			return nil
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
		s.Equal(1, onInvoiceIssuedCnt)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})
}

func (s *LineEngineTestSuite) TestOnInvoiceIssuedFailureTransitionsToRetryableIssuingState() {
	var (
		ctx                = context.Background()
		namespace          = s.GetUniqueNamespace("ns-line-engine-on-invoice-issued-failed")
		mockEngine         = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		invoice            ombilling.StandardInvoice
		collectionAt       time.Time
		onInvoiceIssuedCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a failing issued hook", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - invoice issued hook failed",
		)
	})

	s.Run("When the invoice is collected and approval hits the failing issued hook", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			onInvoiceIssuedCnt++
			s.Equal(input.Invoice.ID, invoice.ID)
			return errors.New("simulated invoice issued failure")
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)

		s.Equal(ombilling.StandardInvoiceStatusIssuingChargeBookingFailed, invoice.Status)
		s.True(invoice.StatusDetails.Failed)
		s.NotNil(invoice.StatusDetails.AvailableActions.Retry)
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingPending, invoice.StatusDetails.AvailableActions.Retry.ResultingState)
		s.Len(invoice.ValidationIssues, 1)
		s.Equal(ombilling.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
		s.Equal("simulated invoice issued failure", invoice.ValidationIssues[0].Message)
	})

	s.Run("Then retry succeeds without re-finalizing the invoice", func() {
		defer mockEngine.Reset()

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			onInvoiceIssuedCnt++
			s.Equal(input.Invoice.ID, invoice.ID)
			return nil
		}

		var err error
		invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(2, onInvoiceIssuedCnt)
		s.Contains(
			[]ombilling.StandardInvoiceStatus{
				ombilling.StandardInvoiceStatusPaymentProcessingPending,
				ombilling.StandardInvoiceStatusPaid,
			},
			invoice.Status,
		)
	})
}

func (s *LineEngineTestSuite) TestOnPaymentAuthorizedIsCalled() {
	var (
		ctx                    = s.T().Context()
		namespace              = s.GetUniqueNamespace("ns-line-engine-on-payment-authorized")
		mockEngine             = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		customInvoicingApp     = s.SetupCustomInvoicing(namespace).App
		invoice                ombilling.StandardInvoice
		collectionAt           time.Time
		onPaymentAuthorizedCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a payment-authorized hook", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollectionForApp(
			ctx,
			namespace,
			customInvoicingApp.GetID(),
			mockEngine.GetLineEngineType(),
			"UBP - payment authorized hook",
		)
	})

	s.Run("When the invoice is collected, issued, and marked authorized", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		mockEngine.onPaymentAuthorized = func(_ context.Context, input ombilling.OnPaymentAuthorizedInput) error {
			onPaymentAuthorizedCnt++
			s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingBookingAuthorized, input.Invoice.Status)
			s.Equal(invoice.ID, input.Invoice.ID)
			s.Len(input.Lines, 1)
			s.Equal(invoice.ID, input.Lines[0].InvoiceID)
			s.Equal(mockEngine.GetLineEngineType(), input.Lines[0].Engine)
			return nil
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		invoice = s.markInvoiceAuthorized(ctx, invoice.GetInvoiceID())
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)
	})

	s.Run("Then the payment-authorized hook is called once", func() {
		s.Equal(1, onPaymentAuthorizedCnt)
		s.Contains(
			[]ombilling.StandardInvoiceStatus{
				ombilling.StandardInvoiceStatusPaymentProcessingAuthorized,
				ombilling.StandardInvoiceStatusPaid,
			},
			invoice.Status,
		)
	})
}

func (s *LineEngineTestSuite) TestOnPaymentAuthorizedFailureTransitionsToRetryablePaymentState() {
	var (
		ctx                    = s.T().Context()
		namespace              = s.GetUniqueNamespace("ns-line-engine-on-payment-authorized-failed")
		mockEngine             = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		customInvoicingApp     = s.SetupCustomInvoicing(namespace).App
		invoice                ombilling.StandardInvoice
		collectionAt           time.Time
		onPaymentAuthorizedCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a failing payment-authorized hook", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollectionForApp(
			ctx,
			namespace,
			customInvoicingApp.GetID(),
			mockEngine.GetLineEngineType(),
			"UBP - payment authorized hook failed",
		)
	})

	s.Run("When the invoice is collected, issued, and authorization hits the failing hook", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		mockEngine.onPaymentAuthorized = func(_ context.Context, input ombilling.OnPaymentAuthorizedInput) error {
			onPaymentAuthorizedCnt++
			s.Equal(invoice.ID, input.Invoice.ID)
			return errors.New("simulated payment authorized failure")
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		invoice = s.markInvoiceAuthorized(ctx, invoice.GetInvoiceID())

		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingBookingAuthorizedFailed, invoice.Status)
		s.True(invoice.StatusDetails.Failed)
		s.NotNil(invoice.StatusDetails.AvailableActions.Retry)
		s.Len(invoice.ValidationIssues, 1)
		s.Equal(ombilling.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
		s.Equal("simulated payment authorized failure", invoice.ValidationIssues[0].Message)
	})

	s.Run("Then retry succeeds without re-issuing the invoice", func() {
		defer mockEngine.Reset()

		mockEngine.onPaymentAuthorized = func(_ context.Context, input ombilling.OnPaymentAuthorizedInput) error {
			onPaymentAuthorizedCnt++
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		var err error
		invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(2, onPaymentAuthorizedCnt)
		s.Contains(
			[]ombilling.StandardInvoiceStatus{
				ombilling.StandardInvoiceStatusPaymentProcessingAuthorized,
				ombilling.StandardInvoiceStatusPaid,
			},
			invoice.Status,
		)
	})
}

func (s *LineEngineTestSuite) TestOnPaymentSettledIsCalled() {
	var (
		ctx                 = s.T().Context()
		namespace           = s.GetUniqueNamespace("ns-line-engine-on-payment-settled")
		mockEngine          = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		invoice             ombilling.StandardInvoice
		collectionAt        time.Time
		onPaymentSettledCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a payment-settled hook", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - payment settled hook",
		)
	})

	s.Run("When the invoice is collected, issued, and marked paid", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		mockEngine.onPaymentSettled = func(_ context.Context, input ombilling.OnPaymentSettledInput) error {
			onPaymentSettledCnt++
			s.Contains(
				[]ombilling.StandardInvoiceStatus{
					ombilling.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettled,
					ombilling.StandardInvoiceStatusPaymentProcessingBookingSettled,
				},
				input.Invoice.Status,
			)
			s.Equal(invoice.ID, input.Invoice.ID)
			s.Len(input.Lines, 1)
			s.Equal(invoice.ID, input.Lines[0].InvoiceID)
			s.Equal(mockEngine.GetLineEngineType(), input.Lines[0].Engine)
			return nil
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})

	s.Run("Then the payment-settled hook is called once", func() {
		s.Equal(1, onPaymentSettledCnt)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})
}

func (s *LineEngineTestSuite) TestOnPaymentSettledFailureTransitionsToRetryablePaymentState() {
	var (
		ctx                 = s.T().Context()
		namespace           = s.GetUniqueNamespace("ns-line-engine-on-payment-settled-failed")
		mockEngine          = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		invoice             ombilling.StandardInvoice
		collectionAt        time.Time
		onPaymentSettledCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a failing payment-settled hook", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - payment settled hook failed",
		)
	})

	s.Run("When the invoice is collected, issued, and payment settlement hits the failing hook", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		mockEngine.onPaymentSettled = func(_ context.Context, input ombilling.OnPaymentSettledInput) error {
			onPaymentSettledCnt++
			s.Equal(invoice.ID, input.Invoice.ID)
			return errors.New("simulated payment settled failure")
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettledFailed, invoice.Status)

		s.Contains(
			[]ombilling.StandardInvoiceStatus{
				ombilling.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettledFailed,
				ombilling.StandardInvoiceStatusPaymentProcessingBookingSettledFailed,
			},
			invoice.Status,
		)
		s.True(invoice.StatusDetails.Failed)
		s.NotNil(invoice.StatusDetails.AvailableActions.Retry)
		s.NotEmpty(invoice.ValidationIssues)
		s.Equal(ombilling.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
		s.Equal("simulated payment settled failure", invoice.ValidationIssues[0].Message)
	})

	s.Run("Then retry succeeds without restarting payment processing", func() {
		defer mockEngine.Reset()

		mockEngine.onPaymentSettled = func(_ context.Context, input ombilling.OnPaymentSettledInput) error {
			onPaymentSettledCnt++
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		var err error
		invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(2, onPaymentSettledCnt)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})
}

func (s *LineEngineTestSuite) TestOnPaymentSettledIsCalledAfterAuthorization() {
	var (
		ctx                 = s.T().Context()
		namespace           = s.GetUniqueNamespace("ns-line-engine-on-payment-settled-after-authorization")
		mockEngine          = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		customInvoicingApp  = s.SetupCustomInvoicing(namespace).App
		invoice             ombilling.StandardInvoice
		collectionAt        time.Time
		onPaymentSettledCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a payment-settled hook after authorization", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollectionForApp(
			ctx,
			namespace,
			customInvoicingApp.GetID(),
			mockEngine.GetLineEngineType(),
			"UBP - payment settled hook after authorization",
		)
	})

	s.Run("When the invoice is collected, issued, marked authorized, and then marked paid", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		mockEngine.onPaymentSettled = func(_ context.Context, input ombilling.OnPaymentSettledInput) error {
			onPaymentSettledCnt++
			s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingBookingSettled, input.Invoice.Status)
			s.Equal(invoice.ID, input.Invoice.ID)
			s.Len(input.Lines, 1)
			s.Equal(invoice.ID, input.Lines[0].InvoiceID)
			s.Equal(mockEngine.GetLineEngineType(), input.Lines[0].Engine)
			return nil
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		invoice = s.markInvoiceAuthorized(ctx, invoice.GetInvoiceID())
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

		invoice = s.markInvoicePaid(ctx, invoice.GetInvoiceID())
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})

	s.Run("Then the payment-settled hook is called once", func() {
		s.Equal(1, onPaymentSettledCnt)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})
}

func (s *LineEngineTestSuite) TestOnPaymentSettledFailureAfterAuthorizationTransitionsToRetryablePaymentState() {
	var (
		ctx                 = s.T().Context()
		namespace           = s.GetUniqueNamespace("ns-line-engine-on-payment-settled-failed-after-authorization")
		mockEngine          = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		customInvoicingApp  = s.SetupCustomInvoicing(namespace).App
		invoice             ombilling.StandardInvoice
		collectionAt        time.Time
		onPaymentSettledCnt int
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	s.Run("Given a draft invoice waiting for collection with a failing payment-settled hook after authorization", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollectionForApp(
			ctx,
			namespace,
			customInvoicingApp.GetID(),
			mockEngine.GetLineEngineType(),
			"UBP - payment settled hook failed after authorization",
		)
	})

	s.Run("When the invoice is collected, issued, marked authorized, and payment settlement hits the failing hook", func() {
		defer mockEngine.Reset()

		mockEngine.onCollectionCompleted = func(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
			lines := input.Lines
			for _, stdLine := range lines {
				if stdLine.UsageBased == nil {
					stdLine.UsageBased = &ombilling.UsageBasedLine{}
				}

				stdLine.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
				stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			}

			return lines, nil
		}

		mockEngine.onInvoiceIssued = func(_ context.Context, input ombilling.OnInvoiceIssuedInput) error {
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		mockEngine.onPaymentSettled = func(_ context.Context, input ombilling.OnPaymentSettledInput) error {
			onPaymentSettledCnt++
			s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingBookingSettled, input.Invoice.Status)
			s.Equal(invoice.ID, input.Invoice.ID)
			return errors.New("simulated payment settled failure")
		}

		clock.SetTime(collectionAt.Add(time.Minute))

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		invoice = s.markInvoiceAuthorized(ctx, invoice.GetInvoiceID())
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

		invoice = s.markInvoicePaid(ctx, invoice.GetInvoiceID())
		s.Equal(ombilling.StandardInvoiceStatusPaymentProcessingBookingSettledFailed, invoice.Status)
		s.True(invoice.StatusDetails.Failed)
		s.NotNil(invoice.StatusDetails.AvailableActions.Retry)
		s.NotEmpty(invoice.ValidationIssues)
		s.Equal(ombilling.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
		s.Equal("simulated payment settled failure", invoice.ValidationIssues[0].Message)
	})

	s.Run("Then retry succeeds without restarting payment processing", func() {
		defer mockEngine.Reset()

		mockEngine.onPaymentSettled = func(_ context.Context, input ombilling.OnPaymentSettledInput) error {
			onPaymentSettledCnt++
			s.Equal(invoice.ID, input.Invoice.ID)
			return nil
		}

		var err error
		invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(2, onPaymentSettledCnt)
		s.Equal(ombilling.StandardInvoiceStatusPaid, invoice.Status)
	})
}
