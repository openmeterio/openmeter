package subscription_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestMigrationAdditionPreservesIssuedInvoice(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	deps := setup(t, setupConfig{})
	defer deps.cleanup(t)

	// given an issued invoice for the original compute size
	before, issued := createBilledComputeSubscription(t, deps)
	target := addComputeSizeToPlan(t, deps, before)

	// when migration adds a compute size in the middle of the month
	clock.FreezeTime(start.Add(14 * 24 * time.Hour))
	after := migrateAndSyncSubscription(t, deps, before, target)

	// then repeated sync leaves the issued invoice alone and bills only the new item
	assertIssuedInvoiceUnchanged(t, deps, issued)
	assertOnlyAddedComputeNeedsBilling(t, deps, after)
	require.Equal(t, before.Subscription.ID, after.Subscription.ID)
}

func monthlyComputeRateCard(t *testing.T, key string, amount int64) productcatalog.RateCard {
	t.Helper()

	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:  key,
			Name: key,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(amount),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
	}
}

func createBilledComputeSubscription(t *testing.T, deps testDeps) (subscription.SubscriptionView, billing.StandardInvoice) {
	t.Helper()

	ctx := t.Context()
	_, err := deps.billingService.CreateProfile(ctx, minimalCreateProfileInputTemplate(deps.sandboxApp.GetID()))
	require.NoError(t, err)

	customer := createUSDSubscriptionCustomer(t, deps, "test-namespace", "compute-customer")

	input := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, monthlyComputeRateCard(t, "compute-l", 100)).Build()
	input.ProRatingConfig = productcatalog.ProRatingConfig{
		Enabled: true,
		Mode:    productcatalog.ProRatingModeProratePrices,
	}
	p := deps.PlanHelper.CreatePlan(t, input)

	before, err := deps.subscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:  customer.Namespace,
		CustomerID: customer.ID,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
		},
	}, p)
	require.NoError(t, err)

	require.NoError(t, deps.subscriptionSyncService.SyncByView(ctx, before, clock.Now()))

	clock.FreezeTime(clock.Now().Add(24 * time.Hour))
	drafts, err := deps.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	require.NoError(t, err)
	require.Len(t, drafts, 1)

	issued, err := deps.billingService.ApproveInvoice(ctx, drafts[0].GetInvoiceID())
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusPaid, issued.Status)

	return before, issued
}

func addComputeSizeToPlan(t *testing.T, deps testDeps, before subscription.SubscriptionView) subscription.Plan {
	t.Helper()

	original, err := deps.PlanService.GetPlan(t.Context(), plan.GetPlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: before.Subscription.Namespace,
			ID:        before.Subscription.PlanRef.Id,
		},
	})
	require.NoError(t, err)

	target := original.AsProductCatalogPlan()
	target.Phases[0].RateCards = append(target.Phases[0].RateCards, monthlyComputeRateCard(t, "compute-xl", 150))

	return deps.PlanHelper.CreatePlan(t, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: original.Namespace},
		Plan:            target,
	})
}

func migrateAndSyncSubscription(t *testing.T, deps testDeps, before subscription.SubscriptionView, target subscription.Plan) subscription.SubscriptionView {
	t.Helper()

	response, err := deps.pcSubscriptionService.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID:            before.Subscription.NamespacedID,
		TargetVersion: lo.ToPtr(target.ToCreateSubscriptionPlanInput().Plan.Version),
	})
	require.NoError(t, err)

	for range 2 {
		require.NoError(t, deps.subscriptionSyncService.SyncByView(t.Context(), response.Next, clock.Now()))
	}

	return response.Next
}

func assertIssuedInvoiceUnchanged(t *testing.T, deps testDeps, issued billing.StandardInvoice) {
	t.Helper()

	persisted, err := deps.billingService.GetStandardInvoiceById(t.Context(), billing.GetStandardInvoiceByIdInput{
		Invoice: issued.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	require.NoError(t, err)

	require.Equal(t, issued.UpdatedAt, persisted.UpdatedAt)
	require.Equal(t, issued.Totals, persisted.Totals)
	require.Equal(t, issued.ValidationIssues, persisted.ValidationIssues)
	require.Equal(t, issued.Lines, persisted.Lines)
}

func assertOnlyAddedComputeNeedsBilling(t *testing.T, deps testDeps, after subscription.SubscriptionView) {
	t.Helper()

	invoices, err := deps.billingService.ListGatheringInvoices(t.Context(), billing.ListGatheringInvoicesInput{
		Namespace: after.Subscription.Namespace,
		Customers: []string{after.Subscription.CustomerId},
		Page:      pagination.Page{PageSize: 10, PageNumber: 1},
		Expand:    billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
	})
	require.NoError(t, err)
	require.Len(t, invoices.Items, 1)

	// Sync also creates next month's lines; only inspect the month of migration.
	monthEnd := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	lines := lo.Filter(invoices.Items[0].Lines.OrEmpty(), func(line billing.GatheringLine, _ int) bool {
		return line.ServicePeriod.From.Before(monthEnd)
	})
	require.Len(t, lines, 1)
	require.Equal(t, "compute-xl", lines[0].Name)
}
