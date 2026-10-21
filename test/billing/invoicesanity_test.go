package billing

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/log"
)

type InvoiceSanityTestSuite struct {
	BaseSuite
	appFactory *sanityAppFactory
}

func TestInvoiceSanity(t *testing.T) {
	// The billing harness shares a clock and mutable app factory across subtests.
	suite.Run(t, new(InvoiceSanityTestSuite))
}

func (s *InvoiceSanityTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.appFactory = &sanityAppFactory{AppFactory: s.SandboxApp.Factory}
	listing := appsandbox.MarketplaceListing
	listing.Type = app.AppTypeStripe
	s.Require().NoError(s.AppService.RegisterMarketplaceListing(app.RegistryItem{
		Listing: listing,
		Factory: s.appFactory,
	}))
}

type sanityTestMode struct {
	name     string
	enforced bool
}

type sanityServiceInput struct {
	EnforceInvoiceSanityChecks bool
	Logger                     *slog.Logger
}

// configureSanityService exercises constructor configuration against the same
// database, including recovery after enforcement is disabled between attempts.
func (s *InvoiceSanityTestSuite) configureSanityService(input sanityServiceInput) {
	s.T().Helper()
	logger := input.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	service, err := billingservice.New(billingservice.Config{
		EnforceInvoiceSanityChecks: input.EnforceInvoiceSanityChecks,
		Adapter:                    s.BillingAdapter,
		SequenceService:            s.SequenceService,
		LegacyBillingLineEngine:    s.LegacyBillingLineEngine,
		CustomerService:            s.CustomerService,
		AppService:                 s.AppService,
		TaxCodeService:             s.TaxCodeService,
		FeatureMeterResolver:       s.FeatureMeterResolver,
		Logger:                     logger,
		Publisher:                  eventbus.NewMock(s.T()),
		AdvancementStrategy:        billing.ForegroundAdvancementStrategy,
	})
	s.Require().NoError(err)
	s.BillingService = service
}

func (s *InvoiceSanityTestSuite) createSanityDraft(appMock *sanityAppMock) billing.StandardInvoice {
	s.T().Helper()
	ctx := s.T().Context()
	s.appFactory.appMock = appMock
	namespace := s.GetUniqueNamespace("invoice-sanity")
	appBase, err := s.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace: namespace,
		Name:      "Observed invoicing app",
		Type:      app.AppTypeStripe,
	})
	s.Require().NoError(err)
	s.ProvisionBillingProfile(ctx, namespace, appBase.GetID())
	customer := s.CreateTestCustomer(namespace, "sanity-subject")

	return s.CreateDraftInvoice(s.T(), ctx, DraftInvoiceInput{
		Namespace: namespace,
		Customer:  customer,
	})
}

func (s *InvoiceSanityTestSuite) reloadSanityInvoice(invoice billing.StandardInvoice) billing.StandardInvoice {
	s.T().Helper()
	reloaded, err := s.BillingService.GetStandardInvoiceById(s.T().Context(), billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Equal(invoice.Status, reloaded.Status)
	s.Equal(invoice.ValidationIssues, reloaded.ValidationIssues)

	return reloaded
}

// persistSanitySnapshot seeds a historical inconsistency or repairs its stored
// detail through the real adapter. No public edit operation can request corrupt
// calculated totals, and retry must consume the stored snapshot without rerating.
func (s *InvoiceSanityTestSuite) persistSanitySnapshot(invoice billing.StandardInvoice) billing.StandardInvoice {
	s.T().Helper()
	s.Require().NoError(invoicecalc.RecalculateTotals(&invoice))
	invoice.StatusDetails = billing.StandardInvoiceStatusDetails{}
	updated, err := s.BillingAdapter.UpdateStandardInvoice(s.T().Context(), invoice)
	s.Require().NoError(err)

	return updated
}

func (s *InvoiceSanityTestSuite) TestFinalizationHandoffAndRetry() {
	for _, mode := range []sanityTestMode{{name: "enforced", enforced: true}, {name: "log only"}, {name: "enforced again", enforced: true}} {
		s.Run(mode.name, func() {
			// Given a valid draft whose engine loses discount detail during finalization.
			clock.FreezeTime(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: mode.enforced})
			appMock := newSanityAppMock(s.T())
			appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
			invoice := s.createSanityDraft(appMock)
			s.Require().Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
			s.Require().NoError(stdinvoicesanity.Validate(invoice))
			engine := &sanityLineEngine{
				LineEngine: s.LegacyBillingLineEngine,
				corruptAt:  billing.StandardInvoiceStatusIssuingLineFinalization,
			}
			s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
			s.Require().NoError(s.BillingService.RegisterLineEngine(engine))
			if !mode.enforced {
				for _, method := range []string{"UpsertStandardInvoice", "FinalizeStandardInvoice"} {
					appMock.On(method, mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
						return invoice.Status == billing.StandardInvoiceStatusIssuingSyncing && stdinvoicesanity.Validate(invoice) != nil
					})).Return(nil).Once()
				}
			}

			// When approval finalizes the lines and attempts to hand them to the app.
			invoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)

			// Then enforcement controls the handoff, and the result survives a reload.
			s.Equal(1, engine.finalizationCalls)
			if mode.enforced {
				s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
				appMock.AssertNumberOfCalls(s.T(), "ValidateStandardInvoice", 1)
				appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 1)
				appMock.AssertNotCalled(s.T(), "FinalizeStandardInvoice", mock.Anything)
				s.Require().Len(invoice.ValidationIssues, 2)
				s.True(invoice.StatusDetails.Failed)
				s.NotNil(invoice.StatusDetails.AvailableActions.Retry)
			} else {
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
				appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 2)
				appMock.AssertNumberOfCalls(s.T(), "FinalizeStandardInvoice", 1)
				s.Empty(invoice.ValidationIssues)
			}

			invoice = s.reloadSanityInvoice(invoice)

			if mode.enforced {
				// When the same stored inconsistency is retried, completed line work is not replayed.
				invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
				s.Equal(1, engine.finalizationCalls)
				appMock.AssertNumberOfCalls(s.T(), "ValidateStandardInvoice", 1)
				appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 1)
				appMock.AssertNotCalled(s.T(), "FinalizeStandardInvoice", mock.Anything)
				s.Require().Len(invoice.ValidationIssues, 2)
				s.reloadSanityInvoice(invoice)
			}
		})
	}
}

func (s *InvoiceSanityTestSuite) TestIssuingRecovery() {
	for _, recovery := range []string{"repair", "disable enforcement"} {
		s.Run(recovery, func() {
			// Given a persisted issuing failure and an unrelated warning.
			clock.FreezeTime(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: true})
			appMock := newSanityAppMock(s.T())
			appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
			invoice := s.createSanityDraft(appMock)
			engine := &sanityLineEngine{
				LineEngine: s.LegacyBillingLineEngine,
				corruptAt:  billing.StandardInvoiceStatusIssuingLineFinalization,
			}
			s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
			s.Require().NoError(s.BillingService.RegisterLineEngine(engine))
			invoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)
			s.Require().Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
			invoice = s.reloadSanityInvoice(invoice)
			warning := billing.NewValidationWarning("unrelated", "unrelated warning")
			warning.Component = "another.component"
			invoice.ValidationIssues = append(invoice.ValidationIssues, warning)

			// When the stored detail is repaired or the service is restarted in log-only mode.
			if recovery == "repair" {
				for _, line := range invoice.Lines.OrEmpty() {
					for idx := range line.DetailedLines {
						line.DetailedLines[idx].AmountDiscounts = billing.AmountLineDiscountsManaged{{
							AmountLineDiscount: billing.AmountLineDiscount{
								LineDiscountBase: billing.LineDiscountBase{Reason: billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{})},
								Amount:           line.DetailedLines[idx].Totals.DiscountsTotal,
							},
						}}
					}
				}
			} else {
				s.configureSanityService(sanityServiceInput{})
			}

			invoice = s.persistSanitySnapshot(invoice)
			for _, method := range []string{"UpsertStandardInvoice", "FinalizeStandardInvoice"} {
				appMock.On(method, mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
					return recovery != "repair" || stdinvoicesanity.Validate(invoice) == nil
				})).Return(nil).Once()
			}

			invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())

			// Then app calls resume, stale sanity issues clear, and unrelated warnings survive.
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
			appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 2)
			appMock.AssertNumberOfCalls(s.T(), "FinalizeStandardInvoice", 1)
			s.False(invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent))
			s.Require().Len(invoice.ValidationIssues, 1)
			s.Equal(warning.Code, invoice.ValidationIssues[0].Code)
			s.Equal(1, engine.finalizationCalls)
			s.reloadSanityInvoice(invoice)
		})
	}
}

func (s *InvoiceSanityTestSuite) TestDraftValidationHandoff() {
	for _, enforced := range []bool{false, true} {
		s.Run(map[bool]string{false: "log only", true: "enforced"}[enforced], func() {
			// Given a line engine that loses the discount breakdown during collection.
			s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: enforced})
			engine := &sanityLineEngine{
				LineEngine: s.LegacyBillingLineEngine,
				corruptAt:  billing.StandardInvoiceStatusDraftUpdating,
			}
			s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
			s.Require().NoError(s.BillingService.RegisterLineEngine(engine))
			appMock := newSanityAppMock(s.T())
			if !enforced {
				appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
				appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
			}

			// When the service creates, collects, and validates a draft.
			invoice := s.createSanityDraft(appMock)

			// Then enforced validation prevents both app validation and draft upsert.
			if enforced {
				s.Equal(billing.StandardInvoiceStatusDraftInvalid, invoice.Status)
				appMock.AssertNotCalled(s.T(), "ValidateStandardInvoice", mock.Anything)
				appMock.AssertNotCalled(s.T(), "UpsertStandardInvoice", mock.Anything)
				s.True(invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent))
			} else {
				s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
				appMock.AssertNumberOfCalls(s.T(), "ValidateStandardInvoice", 1)
				appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 1)
				s.Empty(invoice.ValidationIssues)
			}

			s.reloadSanityInvoice(invoice)
		})
	}
}

func (s *InvoiceSanityTestSuite) TestDraftSyncRetryPreservesAppIssues() {
	// Given an app sync failure followed by a historical inconsistency in the stored invoice.
	s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: true})
	appMock := newSanityAppMock(s.T())
	appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
	appMock.On("UpsertStandardInvoice", mock.Anything).Return(errors.New("provider unavailable")).Once()
	invoice := s.createSanityDraft(appMock)
	s.Require().Equal(billing.StandardInvoiceStatusDraftSyncFailed, invoice.Status)
	s.Require().Len(invoice.ValidationIssues, 1)
	appComponent := invoice.ValidationIssues[0].Component
	invoice = s.reloadSanityInvoice(invoice)
	corruptSanityDiscounts(invoice.Lines.OrEmpty())
	invoice = s.persistSanitySnapshot(invoice)

	// When retry revalidates the saved draft before attempting synchronization.
	invoice, err := s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())

	// Then the app is skipped and its prior issue is preserved alongside the sanity issues.
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftInvalid, invoice.Status)
	appMock.AssertNumberOfCalls(s.T(), "ValidateStandardInvoice", 1)
	appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 1)
	appMock.AssertNotCalled(s.T(), "FinalizeStandardInvoice", mock.Anything)
	s.True(invoice.ValidationIssues.HasComponent(appComponent))
	s.True(invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent))
	invoice = s.reloadSanityInvoice(invoice)

	// When enforcement is disabled, a real provider failure keeps its app classification.
	s.configureSanityService(sanityServiceInput{})
	appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
	appMock.On("UpsertStandardInvoice", mock.Anything).Return(errors.New("provider unavailable")).Once()
	invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftSyncFailed, invoice.Status)
	appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 2)
	s.Require().Len(invoice.ValidationIssues, 1)
	s.Equal(appComponent, invoice.ValidationIssues[0].Component)
	s.Equal("provider unavailable", invoice.ValidationIssues[0].Message)
	s.reloadSanityInvoice(invoice)
}

func (s *InvoiceSanityTestSuite) TestDeletionHandoff() {
	for _, enforced := range []bool{false, true} {
		s.Run(map[bool]string{false: "log only", true: "enforced"}[enforced], func() {
			// Given an inconsistent draft, blocked by sanity validation when enforced.
			s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: enforced})
			engine := &sanityLineEngine{
				LineEngine: s.LegacyBillingLineEngine,
				corruptAt:  billing.StandardInvoiceStatusDraftUpdating,
			}
			s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
			s.Require().NoError(s.BillingService.RegisterLineEngine(engine))
			appMock := newSanityAppMock(s.T())
			appMock.postAdvance = true
			if !enforced {
				appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
				appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
				appMock.On("PostAdvanceStandardInvoiceHook", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
					return invoice.Status == billing.StandardInvoiceStatusDraftWaitingAutoApproval
				})).Return(nil).Once()
			}

			invoice := s.createSanityDraft(appMock)
			s.Require().Error(stdinvoicesanity.Validate(invoice))
			if enforced {
				s.Require().Equal(billing.StandardInvoiceStatusDraftInvalid, invoice.Status)
				s.Require().True(invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent))
			}

			appMock.On("DeleteStandardInvoice", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return stdinvoicesanity.Validate(invoice) != nil && !invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent)
			})).Return(nil).Once()
			appMock.On("PostAdvanceStandardInvoiceHook", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return invoice.Status == billing.StandardInvoiceStatusDeleted && stdinvoicesanity.Validate(invoice) != nil
			})).Return(nil).Once()

			t := s.T()
			handler := log.NewMockHandler()
			handler.Test(t)
			t.Cleanup(func() { handler.AssertExpectations(t) })
			handler.On("Enabled", mock.Anything, mock.Anything).Return(true).Maybe()
			handler.On("Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
				return record.Level != slog.LevelError
			})).Return(nil).Maybe()
			s.configureSanityService(sanityServiceInput{
				EnforceInvoiceSanityChecks: enforced,
				Logger:                     slog.New(handler),
			})

			// When system deletion preserves the inconsistent lines and cleans up the app.
			invoice, err := s.BillingService.DeleteInvoice(s.T().Context(), billing.DeleteInvoiceInput{
				Invoice:        invoice.GetInvoiceID(),
				DeletionSource: billing.ChangeSourceSystem,
			})

			// Then deletion and its post-advance hook succeed without retaining sanity issues.
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusDeleted, invoice.Status)
			appMock.AssertNumberOfCalls(s.T(), "DeleteStandardInvoice", 1)
			s.Empty(invoice.ValidationIssues)
			s.reloadSanityInvoice(invoice)
		})
	}
}

func (s *InvoiceSanityTestSuite) TestPostAdvanceHandoffAfterIssuance() {
	for _, enforced := range []bool{false, true} {
		s.Run(map[bool]string{false: "log only", true: "enforced"}[enforced], func() {
			// Given consistent app handoffs followed by inconsistent line output during charge booking.
			t := s.T()
			handler := log.NewMockHandler()
			handler.Test(t)
			t.Cleanup(func() { handler.AssertExpectations(t) })
			handler.On("Enabled", mock.Anything, mock.Anything).Return(true).Maybe()
			handler.On("Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
				return record.Level != slog.LevelError
			})).Return(nil).Maybe()
			s.configureSanityService(sanityServiceInput{
				EnforceInvoiceSanityChecks: enforced,
				Logger:                     slog.New(handler),
			})
			appMock := newSanityAppMock(s.T())
			appMock.postAdvance = true
			appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("PostAdvanceStandardInvoiceHook", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return invoice.Status == billing.StandardInvoiceStatusDraftWaitingAutoApproval
			})).Return(nil).Once()
			invoice := s.createSanityDraft(appMock)
			engine := &sanityLineEngine{
				LineEngine: s.LegacyBillingLineEngine,
				corruptAt:  billing.StandardInvoiceStatusIssuingChargeBooking,
			}
			s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
			s.Require().NoError(s.BillingService.RegisterLineEngine(engine))

			appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("FinalizeStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("PostAdvanceStandardInvoiceHook", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return invoice.Status == billing.StandardInvoiceStatusPaymentProcessingPending && stdinvoicesanity.Validate(invoice) != nil
			})).Return(nil).Once()

			// When issuing reaches the post-advance hook.
			invoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())

			// Then the issued invoice reaches the hook without sanity issues or error logs in either mode.
			s.Require().NoError(err)
			appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 2)
			appMock.AssertNumberOfCalls(s.T(), "FinalizeStandardInvoice", 1)
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
			appMock.AssertNumberOfCalls(s.T(), "PostAdvanceStandardInvoiceHook", 2)
			s.Empty(invoice.ValidationIssues)
			s.Require().Error(stdinvoicesanity.Validate(invoice))
			s.reloadSanityInvoice(invoice)

			// When payment progresses, later hooks also remain callable without rechecking totals.
			appMock.On("PostAdvanceStandardInvoiceHook", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return invoice.Status == billing.StandardInvoiceStatusPaymentProcessingAuthorized && stdinvoicesanity.Validate(invoice) != nil
			})).Return(nil).Once()
			invoice, err = s.BillingService.PaymentAuthorized(t.Context(), invoice.GetInvoiceID())

			// Then payment progresses without replaying invoice finalization or adding sanity issues.
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)
			s.Empty(invoice.ValidationIssues)
			appMock.AssertNumberOfCalls(s.T(), "PostAdvanceStandardInvoiceHook", 3)
			appMock.AssertNumberOfCalls(s.T(), "FinalizeStandardInvoice", 1)
			s.reloadSanityInvoice(invoice)
		})
	}
}

func (s *InvoiceSanityTestSuite) TestReadinessAndStatusInspection() {
	for _, status := range []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusDraftSyncing, billing.StandardInvoiceStatusIssuingSyncing} {
		s.Run(string(status), func() {
			// Given an async app waiting for confirmation and a subsequently corrupted stored invoice.
			s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: true})
			appMock := newSanityAppMock(s.T())
			appMock.async = true
			appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
			appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
			draftReadiness := appMock.On("CanDraftSyncAdvance", mock.Anything).Return(status == billing.StandardInvoiceStatusIssuingSyncing, nil).Maybe()
			issuingReadiness := appMock.On("CanIssuingSyncAdvance", mock.Anything).Return(false, nil).Maybe()
			invoice := s.createSanityDraft(appMock)
			if status == billing.StandardInvoiceStatusIssuingSyncing {
				appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
				appMock.On("FinalizeStandardInvoice", mock.Anything).Return(nil).Once()
				var err error
				invoice, err = s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
				s.Require().NoError(err)
			}

			s.Require().Equal(status, invoice.Status)
			inconsistentInvoice, err := invoice.Clone()
			s.Require().NoError(err)
			corruptSanityDiscounts(inconsistentInvoice.Lines.OrEmpty())
			invoice = s.persistSanitySnapshot(inconsistentInvoice)

			draftReadiness.Unset()
			issuingReadiness.Unset()

			method := "CanDraftSyncAdvance"
			if status == billing.StandardInvoiceStatusIssuingSyncing {
				method = "CanIssuingSyncAdvance"
			}

			// When status prediction evaluates app readiness, sanity validation is disabled.
			prediction := appMock.On(method, mock.Anything).Run(func(args mock.Arguments) {
				s.Require().Error(stdinvoicesanity.Validate(args.Get(0).(billing.StandardInvoice)))
			}).Return(false, nil)
			invoice = s.reloadSanityInvoice(invoice)
			appMock.AssertCalled(s.T(), method, mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return stdinvoicesanity.Validate(invoice) != nil
			}))
			prediction.Unset()

			// Cache the computed details to isolate the real readiness check from read-time prediction.
			invoice, err = s.BillingAdapter.UpdateStandardInvoice(s.T().Context(), invoice)
			s.Require().NoError(err)
			_, err = s.BillingService.AdvanceInvoice(s.T().Context(), invoice.GetInvoiceID())

			// Then enforcement still blocks the actual app call without persisting issues or changing state.
			s.ErrorIs(err, billing.ErrInvoiceCannotAdvance)
			reloaded := s.reloadSanityInvoice(invoice)
			s.Empty(reloaded.ValidationIssues)

			// When the service flag is false, real readiness calls use reporting-only mode.
			s.configureSanityService(sanityServiceInput{})

			appMock.On(method, mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return stdinvoicesanity.Validate(invoice) != nil
			})).Return(false, nil)
			s.reloadSanityInvoice(invoice)
			_, err = s.BillingService.AdvanceInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.ErrorIs(err, billing.ErrInvoiceCannotAdvance)
			appMock.AssertCalled(s.T(), method, mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
				return stdinvoicesanity.Validate(invoice) != nil
			}))
			s.reloadSanityInvoice(invoice)
		})
	}
}

func (s *InvoiceSanityTestSuite) TestPartialReadsSkipSanityPrediction() {
	for _, mode := range []sanityTestMode{{name: "reporting only"}, {name: "enforced", enforced: true}} {
		s.Run(mode.name, func() {
			// Given a valid custom-invoicing draft whose actions depend on readiness guards.
			t := s.T()
			ctx := t.Context()
			clock.FreezeTime(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			handler := log.NewMockHandler()
			handler.Test(t)
			t.Cleanup(func() { handler.AssertExpectations(t) })
			handler.On("Enabled", mock.Anything, mock.Anything).Return(true).Maybe()
			handler.On("Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
				return record.Level != slog.LevelError
			})).Return(nil).Maybe()
			s.configureSanityService(sanityServiceInput{
				EnforceInvoiceSanityChecks: mode.enforced,
				Logger:                     slog.New(handler),
			})
			namespace := s.GetUniqueNamespace("sanity-partial-read")
			customApp := s.SetupCustomInvoicing(namespace).App
			s.ProvisionBillingProfile(ctx, namespace, customApp.GetID())
			customer := s.CreateTestCustomer(namespace, "partial-read-subject")
			invoice := s.CreateDraftInvoice(t, ctx, DraftInvoiceInput{
				Namespace: namespace,
				Customer:  customer,
			})
			s.Require().Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
			s.Require().NoError(stdinvoicesanity.Validate(invoice))

			// When the same invoice is read with and without expanded lines, and listed without lines.
			partial, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: invoice.GetInvoiceID(),
			})
			s.Require().NoError(err)
			expanded, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: invoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			listed, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
				Namespace: namespace,
				IDs:       []string{invoice.ID},
			})
			s.Require().NoError(err)
			s.Require().Len(listed.Items, 1)
			listedInvoice, err := listed.Items[0].AsStandardInvoice()
			s.Require().NoError(err)

			// Then predictions agree without error logs, response expansion changes, or persisted issues.
			s.True(partial.Lines.IsAbsent())
			s.True(listedInvoice.Lines.IsAbsent())
			s.True(expanded.Lines.IsPresent())
			s.Equal(expanded.StatusDetails, partial.StatusDetails)
			s.Equal(expanded.StatusDetails, listedInvoice.StatusDetails)
			s.Require().NotNil(partial.StatusDetails.AvailableActions.Approve)
			s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, partial.StatusDetails.AvailableActions.Approve.ResultingState)
			s.Empty(partial.ValidationIssues)
			s.Empty(listedInvoice.ValidationIssues)
			s.reloadSanityInvoice(invoice)

			// When real issuing receives inconsistent finalized lines, the service mode is restored.
			engine := &sanityLineEngine{
				LineEngine: s.LegacyBillingLineEngine,
				corruptAt:  billing.StandardInvoiceStatusIssuingLineFinalization,
			}
			s.Require().NoError(s.BillingService.DeregisterLineEngine(engine.GetLineEngineType()))
			s.Require().NoError(s.BillingService.RegisterLineEngine(engine))
			handler.On("Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
				return record.Level == slog.LevelError
			})).Return(nil)
			invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())

			// Then real execution reports or enforces sanity failures despite nested status predictions.
			s.Require().NoError(err)
			if mode.enforced {
				s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
				s.True(invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent))
			} else {
				s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
				s.Empty(invoice.ValidationIssues)
			}

			s.reloadSanityInvoice(invoice)
		})
	}
}

// TestMonetarySanityHandoffAndRecovery writes broken invoices through the
// adapter because public edit operations do not allow these inconsistent totals.
// It checks what reaches the app and whether retry recovers after a repair.
func (s *InvoiceSanityTestSuite) TestMonetarySanityHandoffAndRecovery() {
	for _, mode := range []sanityTestMode{{name: "enforced", enforced: true}, {name: "log only"}} {
		s.Run(mode.name, func() {
			for _, tc := range []struct {
				name                string
				issue               billing.ValidationIssue
				recalculatedOnRetry bool
				corrupt             func(*billing.StandardInvoice)
			}{
				{
					name:  "credit totals",
					issue: stdinvoicesanity.ErrCreditTotalMismatch,
					corrupt: func(invoice *billing.StandardInvoice) {
						invoice.Lines.OrEmpty()[0].DetailedLines[0].CreditsApplied = billing.CreditsApplied{{
							Amount:              alpacadecimal.NewFromInt(1),
							CreditRealizationID: "sanity-test-credit",
						}}
					},
				},

				{
					name: "credit conservation", issue: stdinvoicesanity.ErrCreditConservationMismatch,
					corrupt: func(invoice *billing.StandardInvoice) {
						invoice.Lines.OrEmpty()[0].CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(1), CreditRealizationID: "sanity-test-credit"}}
					},
				},
				{
					name: "calculated precision", issue: stdinvoicesanity.ErrTotalsPrecision,
					corrupt: func(invoice *billing.StandardInvoice) {
						detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
						detail.Totals.Amount = detail.Totals.Amount.Add(alpacadecimal.RequireFromString("0.001"))
					},
				},
				{
					name:  "gross amounts",
					issue: stdinvoicesanity.ErrGrossAmountMismatch,
					corrupt: func(invoice *billing.StandardInvoice) {
						detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
						detail.Quantity = detail.Quantity.Add(alpacadecimal.NewFromInt(1))
					},
				},
				{
					name:  "commitment placement",
					issue: stdinvoicesanity.ErrCommitmentPlacementMismatch,
					corrupt: func(invoice *billing.StandardInvoice) {
						invoice.Lines.OrEmpty()[0].DetailedLines[0].Totals.ChargesTotal = alpacadecimal.NewFromInt(1)
					},
				},
				{
					name:  "total arithmetic",
					issue: stdinvoicesanity.ErrTotalArithmeticMismatch,
					corrupt: func(invoice *billing.StandardInvoice) {
						detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
						detail.Totals.Total = detail.Totals.Total.Add(alpacadecimal.NewFromInt(1))
					},
				},
				{
					name:  "parent aggregation",
					issue: stdinvoicesanity.ErrLineTotalsMismatch,
					corrupt: func(invoice *billing.StandardInvoice) {
						line := invoice.Lines.OrEmpty()[0]
						line.Totals.Amount = line.Totals.Amount.Add(alpacadecimal.NewFromInt(1))
					},
				},
				{
					name:                "invoice service period",
					issue:               stdinvoicesanity.ErrInvoicePeriodMismatch,
					recalculatedOnRetry: true,
					corrupt:             func(invoice *billing.StandardInvoice) { invoice.Period.To = invoice.Period.To.Add(time.Hour) },
				},
				{
					name:                "invoice aggregation",
					issue:               stdinvoicesanity.ErrInvoiceTotalsMismatch,
					recalculatedOnRetry: true,
					corrupt: func(invoice *billing.StandardInvoice) {
						invoice.Totals.Amount = invoice.Totals.Amount.Add(alpacadecimal.NewFromInt(1))
						invoice.Totals.Total = invoice.Totals.CalculateTotal()
					},
				},
			} {
				s.Run(tc.name, func() {
					// Given a failed app sync, followed by an error in the stored invoice.
					s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: mode.enforced})
					appMock := newSanityAppMock(s.T())
					appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
					appMock.On("UpsertStandardInvoice", mock.Anything).Return(errors.New("provider unavailable")).Once()
					original := s.createSanityDraft(appMock)
					s.Require().Equal(billing.StandardInvoiceStatusDraftSyncFailed, original.Status)
					s.Require().NoError(stdinvoicesanity.Validate(original))
					corrupted, err := original.Clone()
					s.Require().NoError(err)
					tc.corrupt(&corrupted)
					corrupted.StatusDetails = billing.StandardInvoiceStatusDetails{}
					// Use the adapter directly to keep the broken invoice totals and period;
					// persistSanitySnapshot would recalculate them.
					corrupted, err = s.BillingAdapter.UpdateStandardInvoice(s.T().Context(), corrupted)
					s.Require().NoError(err)
					corrupted = s.reloadSanityInvoice(corrupted)
					s.Require().ErrorIs(stdinvoicesanity.Validate(corrupted), tc.issue)

					if !mode.enforced || tc.recalculatedOnRetry {
						appMock.On("ValidateStandardInvoice", mock.MatchedBy(func(invoice billing.StandardInvoice) bool {
							// Retry recalculates invoice totals and period before calling the app.
							if tc.recalculatedOnRetry {
								return stdinvoicesanity.Validate(invoice) == nil
							}

							return errors.Is(stdinvoicesanity.Validate(invoice), tc.issue)
						})).Return(nil).Once()
						appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
					}

					// When we retry the invoice before sending it to the app.
					invoice, err := s.BillingService.RetryInvoice(s.T().Context(), corrupted.GetInvoiceID())
					s.Require().NoError(err)

					// Then remaining issues block the app only when enforcement is enabled.
					if !mode.enforced || tc.recalculatedOnRetry {
						s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
						s.False(invoice.ValidationIssues.HasComponent(stdinvoicesanity.ValidationComponent))
						if tc.recalculatedOnRetry {
							s.Require().NoError(stdinvoicesanity.Validate(invoice))
							s.Equal(original.Totals, invoice.Totals)
						}

						appMock.AssertNumberOfCalls(s.T(), "ValidateStandardInvoice", 2)
						appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 2)
						s.reloadSanityInvoice(invoice)

						return
					}

					s.Equal(billing.StandardInvoiceStatusDraftInvalid, invoice.Status)
					appMock.AssertNumberOfCalls(s.T(), "ValidateStandardInvoice", 1)
					appMock.AssertNumberOfCalls(s.T(), "UpsertStandardInvoice", 1)
					invoice = s.reloadSanityInvoice(invoice)
					found := false
					for _, issue := range invoice.ValidationIssues {
						if issue.Code == tc.issue.Code && issue.Component == stdinvoicesanity.ValidationComponent {
							found = true
							s.Equal(billing.ValidationIssueSeverityCritical, issue.Severity)
							s.NotEmpty(issue.Path)
							s.NotEmpty(issue.Attributes["expected"])
							s.NotEmpty(issue.Attributes["actual"])
						}
					}

					s.Require().True(found, "new sanity issue must survive persistence")

					// Restore the original values, then retry to clear the issue and resume sync.
					// Use the latest DBState so the adapter sees what changed since the failure.
					for _, line := range original.Lines.OrEmpty() {
						line.DBState = invoice.Lines.GetByID(line.ID).DBState
					}

					original.Status = invoice.Status
					original.ValidationIssues = invoice.ValidationIssues
					original.StatusDetails = billing.StandardInvoiceStatusDetails{}
					repaired, err := s.BillingAdapter.UpdateStandardInvoice(s.T().Context(), original)
					s.Require().NoError(err)
					s.Require().NoError(stdinvoicesanity.Validate(s.reloadSanityInvoice(repaired)))
					appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
					appMock.On("UpsertStandardInvoice", mock.Anything).Return(nil).Once()
					invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
					s.Require().NoError(err)
					s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
					s.Empty(invoice.ValidationIssues)
					s.Require().NoError(stdinvoicesanity.Validate(invoice))
					s.reloadSanityInvoice(invoice)
				})
			}
		})
	}
}

func (s *InvoiceSanityTestSuite) TestSanityLoggingAtAppHandoff() {
	for _, mode := range []sanityTestMode{{name: "log only"}, {name: "enforced", enforced: true}} {
		s.Run(mode.name, func() {
			for _, testCase := range []struct {
				name         string
				inconsistent bool
			}{{name: "valid invoice"}, {name: "inconsistent invoice", inconsistent: true}} {
				s.Run(testCase.name, func() {
					// Given a draft sync failure and an optional inconsistency in its stored detail.
					s.configureSanityService(sanityServiceInput{EnforceInvoiceSanityChecks: mode.enforced})
					appMock := newSanityAppMock(s.T())
					appMock.On("ValidateStandardInvoice", mock.Anything).Return(nil).Once()
					appMock.On("UpsertStandardInvoice", mock.Anything).Return(errors.New("provider unavailable")).Once()
					invoice := s.createSanityDraft(appMock)
					s.Require().Equal(billing.StandardInvoiceStatusDraftSyncFailed, invoice.Status)
					if testCase.inconsistent {
						inconsistentInvoice, err := invoice.Clone()
						s.Require().NoError(err)
						corruptSanityDiscounts(inconsistentInvoice.Lines.OrEmpty())
						invoice = s.persistSanitySnapshot(inconsistentInvoice)
					}

					t := s.T()
					handler := log.NewMockHandler()
					handler.Test(t)
					t.Cleanup(func() { handler.AssertExpectations(t) })
					handler.On("Enabled", mock.Anything, mock.Anything).Return(true).Maybe()
					handler.On("Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
						return record.Level != slog.LevelError
					})).Return(nil).Maybe()
					if testCase.inconsistent {
						handler.On("Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
							return record.Level == slog.LevelError
						})).Run(func(args mock.Arguments) {
							record := args.Get(1).(slog.Record)
							attrs := make(map[string]slog.Value)
							record.Attrs(func(attr slog.Attr) bool {
								attrs[attr.Key] = attr.Value.Resolve()
								return true
							})
							s.Equal(invoice.ID, attrs["invoice_id"].String())
							s.Equal(invoice.Namespace, attrs["namespace"].String())
							s.Equal(string(billing.StandardInvoiceStatusDraftValidating), attrs["invoice_status"].String())
							s.Equal(mode.enforced, attrs["enforced"].Bool())
							issues, ok := attrs["issues"].Any().(billing.ValidationIssues)
							s.Require().True(ok)
							s.Require().Len(issues, 2)
							for _, issue := range issues {
								s.Equal(stdinvoicesanity.ValidationComponent, issue.Component)
								s.Equal(stdinvoicesanity.ErrDiscountTotalMismatch.Code, issue.Code)
								s.Equal(billing.ValidationIssueSeverityCritical, issue.Severity)
								s.NotEmpty(issue.Path)
								s.NotEmpty(issue.Attributes["detailed_line_id"])
							}
						}).Return(nil).Once()
					}

					s.configureSanityService(sanityServiceInput{
						EnforceInvoiceSanityChecks: mode.enforced,
						Logger:                     slog.New(handler),
					})
					if !testCase.inconsistent || !mode.enforced {
						// The provider rejects validation, keeping this attempt to a single checked handoff.
						appMock.On("ValidateStandardInvoice", mock.Anything).Return(errors.New("provider rejected invoice")).Once()
					}

					// When retry validates the draft before attempting synchronization.
					_, err := s.BillingService.RetryInvoice(t.Context(), invoice.GetInvoiceID())

					// Then failures produce one record containing all issues, while valid invoices log no errors.
					s.Require().NoError(err)
					if !testCase.inconsistent {
						handler.AssertNotCalled(t, "Handle", mock.Anything, mock.MatchedBy(func(record slog.Record) bool {
							return record.Level == slog.LevelError
						}))
					}
				})
			}
		})
	}
}

// sanityLineEngine delegates real rating and lifecycle work, injecting the
// missing-discount incident only at the boundary under test.
type sanityLineEngine struct {
	billing.LineEngine
	corruptAt         billing.StandardInvoiceStatus
	finalizationCalls int
}

func (e *sanityLineEngine) OnCollectionCompleted(ctx context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	lines, err := e.LineEngine.OnCollectionCompleted(ctx, input)
	if err != nil {
		return nil, err
	}

	if e.corruptAt == billing.StandardInvoiceStatusDraftUpdating {
		corruptSanityDiscounts(lines)
	}

	return lines, nil
}

func (e *sanityLineEngine) OnInvoiceFinalizing(ctx context.Context, input billing.OnInvoiceFinalizingInput) (billing.StandardLines, error) {
	e.finalizationCalls++
	lines, err := e.LineEngine.OnInvoiceFinalizing(ctx, input)
	if err != nil {
		return nil, err
	}

	if e.corruptAt == billing.StandardInvoiceStatusIssuingLineFinalization {
		corruptSanityDiscounts(lines)
	}

	return lines, nil
}

func (e *sanityLineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if err := e.LineEngine.OnInvoiceIssued(ctx, input); err != nil {
		return err
	}

	if e.corruptAt == billing.StandardInvoiceStatusIssuingChargeBooking {
		corruptSanityDiscounts(input.Lines)
	}

	return nil
}

func corruptSanityDiscounts(lines billing.StandardLines) {
	for _, line := range lines {
		for idx := range line.DetailedLines {
			detail := &line.DetailedLines[idx]
			detail.AmountDiscounts = nil
			detail.Totals.DiscountsTotal = alpacadecimal.NewFromInt(20)
			detail.Totals.Total = detail.Totals.CalculateTotal()
		}

		line.Totals = line.DetailedLines.SumTotals()
	}
}

type sanityAppMock struct {
	mock.Mock
	async       bool
	postAdvance bool
}

func newSanityAppMock(t *testing.T) *sanityAppMock {
	t.Helper()
	appMock := &sanityAppMock{}
	appMock.Test(t)
	t.Cleanup(func() { appMock.AssertExpectations(t) })

	return appMock
}

type sanityRequiredApp interface {
	app.App
	billing.InvoicingApp
	customerapp.App
}

type sanityInvoiceApp struct {
	sanityRequiredApp
	appMock *sanityAppMock
}

func (a *sanityInvoiceApp) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	if err := a.appMock.MethodCalled("ValidateStandardInvoice", invoice).Error(0); err != nil {
		return err
	}

	return a.sanityRequiredApp.ValidateStandardInvoice(ctx, invoice)
}

func (a *sanityInvoiceApp) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	if err := a.appMock.MethodCalled("UpsertStandardInvoice", invoice).Error(0); err != nil {
		return nil, err
	}

	return a.sanityRequiredApp.UpsertStandardInvoice(ctx, invoice)
}

func (a *sanityInvoiceApp) FinalizeStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.FinalizeStandardInvoiceResult, error) {
	if err := a.appMock.MethodCalled("FinalizeStandardInvoice", invoice).Error(0); err != nil {
		return nil, err
	}

	return a.sanityRequiredApp.FinalizeStandardInvoice(ctx, invoice)
}

func (a *sanityInvoiceApp) DeleteStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	if err := a.appMock.MethodCalled("DeleteStandardInvoice", invoice).Error(0); err != nil {
		return err
	}

	return a.sanityRequiredApp.DeleteStandardInvoice(ctx, invoice)
}

type sanityAsyncInvoiceApp struct{ *sanityInvoiceApp }

func (a *sanityAsyncInvoiceApp) CanDraftSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
	args := a.appMock.MethodCalled("CanDraftSyncAdvance", invoice)
	return args.Bool(0), args.Error(1)
}

func (a *sanityAsyncInvoiceApp) CanIssuingSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
	args := a.appMock.MethodCalled("CanIssuingSyncAdvance", invoice)
	return args.Bool(0), args.Error(1)
}

type sanityHookInvoiceApp struct{ *sanityInvoiceApp }

func (a *sanityHookInvoiceApp) PostAdvanceStandardInvoiceHook(_ context.Context, invoice billing.StandardInvoice) (*billing.PostAdvanceHookResult, error) {
	return nil, a.appMock.MethodCalled("PostAdvanceStandardInvoiceHook", invoice).Error(0)
}

type sanityAppFactory struct {
	app.AppFactory
	appMock *sanityAppMock
}

func (f *sanityAppFactory) NewApp(ctx context.Context, base app.AppBase) (app.App, error) {
	upstream, err := f.AppFactory.NewApp(ctx, base)
	if err != nil {
		return nil, err
	}

	observed := &sanityInvoiceApp{sanityRequiredApp: upstream.(sanityRequiredApp), appMock: f.appMock}
	if f.appMock.async {
		return &sanityAsyncInvoiceApp{sanityInvoiceApp: observed}, nil
	}

	if f.appMock.postAdvance {
		return &sanityHookInvoiceApp{sanityInvoiceApp: observed}, nil
	}

	return observed, nil
}
