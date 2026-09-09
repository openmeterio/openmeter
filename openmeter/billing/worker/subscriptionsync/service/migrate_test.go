package service

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *SubscriptionHandlerTestSuite) TestMigrationAdditionPreservesIssuedInvoice() {
	// given a plan item already billed for the full current month
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()
	s.enableProrating()
	phase := productcatalog.Phase{
		PhaseMeta: s.phaseMeta("first-phase", ""),
		RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key: "compute-l", Name: "compute-l",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(100), PaymentTerm: productcatalog.InAdvancePaymentTerm}),
			},
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}},
	}
	before := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{phase})
	s.Require().NoError(s.Service.SyncByView(ctx, before, clock.Now()))
	clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))
	drafts, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{Customer: s.Customer.GetID(), AsOf: lo.ToPtr(clock.Now())})
	s.Require().NoError(err)
	s.Require().Len(drafts, 1)
	issued, err := s.BillingService.ApproveInvoice(ctx, drafts[0].GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Equal(billing.StandardInvoiceStatusPaid, issued.Status)

	// when a later version adds another item and the same subscription is reconciled
	original, err := s.PlanService.GetPlan(ctx, plan.GetPlanInput{NamespacedID: models.NamespacedID{Namespace: s.Namespace, ID: before.Subscription.PlanRef.Id}})
	s.Require().NoError(err)
	_, err = s.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID:    original.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(clock.Now())},
	})
	s.Require().NoError(err)
	targetPlan := original.AsProductCatalogPlan()
	targetPlan.Phases[0].RateCards = append(targetPlan.Phases[0].RateCards, &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key: "compute-xl", Name: "compute-xl",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(150), PaymentTerm: productcatalog.InAdvancePaymentTerm}),
		},
		BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
	})
	p2, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{NamespacedModel: models.NamespacedModel{Namespace: s.Namespace}, Plan: targetPlan})
	s.Require().NoError(err)
	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
	after, err := s.SubscriptionWorkflowService.MigrateToPlan(ctx, subscriptionworkflow.MigrateSubscriptionWorkflowInput{
		SubscriptionID: before.Subscription.NamespacedID,
		Plan:           &plansubscription.Plan{Plan: p2.AsProductCatalogPlan(), Ref: &p2.NamespacedID},
		Timing:         subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.Service.SyncByView(ctx, after, clock.Now()))
	s.Require().NoError(s.Service.SyncByView(ctx, after, clock.Now()))

	// then the issued invoice is untouched and only the added item needs billing
	// in the current month; normal lookahead may also provision next month's lines
	persisted, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: issued.GetInvoiceID(), Expand: billing.StandardInvoiceExpandAll})
	s.Require().NoError(err)
	s.Equal(issued.UpdatedAt, persisted.UpdatedAt)
	s.Equal(issued.Totals, persisted.Totals)
	s.Equal(issued.ValidationIssues, persisted.ValidationIssues)
	s.Equal(issued.Lines, persisted.Lines)
	gathering := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	currentMonth := lo.Filter(gathering.Lines.OrEmpty(), func(line billing.GatheringLine, _ int) bool {
		return line.ServicePeriod.From.Before(s.mustParseTime("2024-02-01T00:00:00Z"))
	})
	s.Require().Len(currentMonth, 1)
	s.Equal("compute-xl", currentMonth[0].Name)
	s.Equal(before.Subscription.ID, after.Subscription.ID)
}
