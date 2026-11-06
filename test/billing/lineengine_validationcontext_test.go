package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	ombilling "github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The sandbox app drives payment from its post-advance hook. A plain error from the settled hook
// still rolls the attempt back there; only a typed issue is persisted, and it keeps the leaf's
// message, code and attributes while the fmt.Errorf wrapper naming the trigger and charge is dropped.
func (s *LineEngineTestSuite) TestOnPaymentSettledFailureThroughSandboxHookOnlyPersistsTypedIssues() {
	const wrapperText = "settling invoice payment for charge[charge-1]"

	var (
		ctx          = s.T().Context()
		namespace    = s.GetUniqueNamespace("ns-line-engine-on-payment-settled-sandbox-hook")
		mockEngine   = &mockCollectionCompletedLineEngine{engineType: ombilling.LineEngineTypeChargeCreditPurchase}
		invoice      ombilling.StandardInvoice
		collectionAt time.Time
		plainLeaf    = errors.New("payment already settled")
		typedLeaf    = payment.ErrPaymentAlreadySettled.WithAttr("charge_id", "charge-1")
	)

	clockBase := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	clock.SetTime(clockBase)
	defer clock.ResetTime()
	defer func() { _ = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) }()
	defer s.MockStreamingConnector.Reset()
	s.registerMockLineEngine(s.T(), mockEngine)
	defer s.unregisterLineEngine(s.T(), mockEngine)

	introspector, ok := s.BillingAdapter.(ValidationIssueIntrospector)
	s.Require().True(ok)

	s.Run("Given a collected draft invoice waiting for approval", func() {
		defer mockEngine.Reset()

		mockEngine.buildStandardInvoiceLines = func(_ context.Context, input ombilling.BuildStandardInvoiceLinesInput) (ombilling.StandardLines, error) {
			return mustAsNewStandardLines(input), nil
		}

		invoice, collectionAt = s.createMeteredDraftInvoiceWaitingForCollection(
			ctx,
			namespace,
			mockEngine.GetLineEngineType(),
			"UBP - settled hook failure via sandbox hook",
		)

		clock.SetTime(collectionAt.Add(time.Minute))

		mockEngine.onCollectionCompleted = stubCollectionCompletedQuantities

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	})

	s.Run("When the settled hook fails with a plain error, approval rolls back", func() {
		defer mockEngine.Reset()

		mockEngine.onInvoiceIssued = func(_ context.Context, _ ombilling.OnInvoiceIssuedInput) error {
			return nil
		}
		mockEngine.onPaymentSettled = func(_ context.Context, _ ombilling.OnPaymentSettledInput) error {
			return fmt.Errorf("%s: %w", wrapperText, plainLeaf)
		}

		_, err := s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().ErrorIs(err, plainLeaf)

		reloaded := s.reloadStandardInvoice(ctx, invoice.GetInvoiceID())
		s.Equal(ombilling.StandardInvoiceStatusDraftWaitingAutoApproval, reloaded.Status)
		s.Empty(reloaded.ValidationIssues)

		rows, err := introspector.IntrospectValidationIssues(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Empty(rows)
	})

	s.Run("When the settled hook fails with a typed issue, the invoice enters a failed payment state", func() {
		defer mockEngine.Reset()

		mockEngine.onInvoiceIssued = func(_ context.Context, _ ombilling.OnInvoiceIssuedInput) error {
			return nil
		}
		mockEngine.onPaymentSettled = func(_ context.Context, _ ombilling.OnPaymentSettledInput) error {
			return fmt.Errorf("%s: %w", wrapperText, typedLeaf)
		}

		var err error
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Contains(
			[]ombilling.StandardInvoiceStatus{
				ombilling.StandardInvoiceStatusPaymentProcessingBookingAuthorizedAndSettledFailed,
				ombilling.StandardInvoiceStatusPaymentProcessingBookingSettledFailed,
			},
			invoice.Status,
		)
		s.True(invoice.StatusDetails.Failed)
	})

	s.Run("Then the persisted issue carries the leaf code and text but not the wrapper context", func() {
		critical := lo.Filter(invoice.ValidationIssues, func(issue ombilling.ValidationIssue, _ int) bool {
			return issue.Severity == ombilling.ValidationIssueSeverityCritical
		})
		s.Require().Len(critical, 1)
		s.Equal(typedLeaf.Message(), critical[0].Message)
		s.NotContains(critical[0].Message, wrapperText)
		s.Equal(string(typedLeaf.Code()), critical[0].Code)
		s.Equal(models.Annotations{
			"charge_id": "charge-1",
			"commonhttp.httpAttributeKey:openmeter.http.status_code": float64(400),
		}, critical[0].Attributes)
		// assert with the outermost component: app.sandbox.invoiceCustomers.post_advance_hook
		s.Equal(ombilling.AppTypeCapabilityToComponent(invoice.Workflow.Apps.Invoicing.GetType(), app.CapabilityTypeInvoiceCustomers, string(ombilling.StandardInvoiceOpPostAdvanceHook)), critical[0].Component)

		rows, err := introspector.IntrospectValidationIssues(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		live := lo.Filter(rows, func(issue ombilling.ValidationIssue, _ int) bool {
			return issue.DeletedAt == nil && issue.Severity == ombilling.ValidationIssueSeverityCritical
		})
		s.Require().Len(live, 1)
		s.Equal(critical[0].Message, live[0].Message)
		s.Equal(critical[0].Code, live[0].Code)
		s.Equal(critical[0].Component, live[0].Component)
		s.Equal(critical[0].Attributes, live[0].Attributes)
	})
}

func (s *LineEngineTestSuite) reloadStandardInvoice(ctx context.Context, id ombilling.InvoiceID) ombilling.StandardInvoice {
	generic, err := s.BillingService.GetInvoiceById(ctx, ombilling.GetInvoiceByIdInput{
		Invoice: id,
		Expand:  ombilling.InvoiceExpandAll,
	})
	s.Require().NoError(err)

	invoice, err := generic.AsStandardInvoice()
	s.Require().NoError(err)

	return invoice
}

func stubCollectionCompletedQuantities(_ context.Context, input ombilling.OnCollectionCompletedInput) (ombilling.StandardLines, error) {
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
