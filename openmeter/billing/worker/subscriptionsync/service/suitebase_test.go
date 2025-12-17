package service

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type SuiteBase struct {
	billingtest.BaseSuite
	billingtest.SubscriptionMixin
	Service *Service
	Adapter subscriptionsync.Adapter

	Namespace               string
	Customer                *customer.Customer
	APIRequestsTotalFeature feature.Feature
}

func (s *SuiteBase) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.SubscriptionMixin.SetupSuite(s.T(), s.GetSubscriptionMixInDependencies())

	adapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
	})
	s.NoError(err)
	s.Adapter = adapter

	service, err := New(Config{
		BillingService:          s.BillingService,
		Logger:                  slog.Default(),
		Tracer:                  noop.NewTracerProvider().Tracer("test"),
		SubscriptionSyncAdapter: adapter,
		SubscriptionService:     s.SubscriptionService,
	})
	s.NoError(err)

	s.Service = service
}

func (s *SuiteBase) BeforeTest(ctx context.Context, suiteName, testName string) {
	s.Namespace = fmt.Sprintf("t-%s-%s-%s", suiteName, testName, ulid.Make().String())

	appSandbox := s.InstallSandboxApp(s.T(), s.Namespace)

	s.ProvisionBillingProfile(ctx, s.Namespace, appSandbox.GetID())

	apiRequestsTotalMeterSlug := "api-requests-total"

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: s.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "API Requests Total",
			},
			Key:           apiRequestsTotalMeterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.NoError(err, "Replacing meters must not return error")

	apiRequestsTotalFeatureKey := "api-requests-total"

	apiRequestsTotalFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: s.Namespace,
		Name:      "api-requests-total",
		Key:       apiRequestsTotalFeatureKey,
		MeterSlug: lo.ToPtr("api-requests-total"),
	})
	s.NoError(err)
	s.APIRequestsTotalFeature = apiRequestsTotalFeature

	customerEntity := s.CreateTestCustomer(s.Namespace, "test")
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	s.Customer = customerEntity
}

func (s *SuiteBase) AfterTest(ctx context.Context, suiteName, testName string) {
	clock.UnFreeze()
	clock.ResetTime()

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
	s.NoError(err, "Replacing meters must not return error")

	s.MockStreamingConnector.Reset()
	s.Service.featureFlags = FeatureFlags{}
}

func (s *SuiteBase) gatheringInvoice(ctx context.Context, namespace string, customerID string) billing.Invoice {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 1, "expected 1 gathering invoice")
	return invoices.Items[0]
}

func (s *SuiteBase) expectNoGatheringInvoice(ctx context.Context, namespace string, customerID string) {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 0)
}

func (s *SuiteBase) enableProrating() {
	s.Service.featureFlags.EnableFlatFeeInAdvanceProrating = true
	s.Service.featureFlags.EnableFlatFeeInArrearsProrating = true
}

func (s *SuiteBase) getLineByChildID(invoice billing.Invoice, childID string) *billing.Line {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			return line
		}
	}

	s.Failf("line not found", "line with child id %s not found", childID)

	return nil
}

func (s *SuiteBase) expectNoLineWithChildID(invoice billing.Invoice, childID string) {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			s.Failf("line found", "line with child id %s found", childID)
		}
	}
}

func (s *SuiteBase) timingImmediate() subscription.Timing {
	return subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	}
}

func (s *SuiteBase) getPhaseByKey(t *testing.T, subsView subscription.SubscriptionView, key string) subscription.SubscriptionPhaseView {
	for _, phase := range subsView.Phases {
		if phase.SubscriptionPhase.Key == key {
			return phase
		}
	}

	t.Fatalf("phase with key %s not found", key)
	return subscription.SubscriptionPhaseView{}
}

type expectedLine struct {
	Matcher          lineMatcher
	Qty              mo.Option[float64]
	Price            mo.Option[*productcatalog.Price]
	Periods          []billing.Period
	InvoiceAt        mo.Option[[]time.Time]
	AdditionalChecks func(line *billing.Line)
}

func (s *SuiteBase) expectLines(invoice billing.Invoice, subscriptionID string, expectedLines []expectedLine) {
	s.T().Helper()

	lines := invoice.Lines.OrEmpty()

	existingLineChildIDs := lo.Map(lines, func(line *billing.Line, _ int) string {
		return lo.FromPtrOr(line.ChildUniqueReferenceID, line.ID)
	})

	expectedLineIds := lo.Flatten(lo.Map(expectedLines, func(expectedLine expectedLine, _ int) []string {
		return expectedLine.Matcher.ChildIDs(subscriptionID)
	}))

	s.ElementsMatch(expectedLineIds, existingLineChildIDs)

	for _, expectedLine := range expectedLines {
		childIDs := expectedLine.Matcher.ChildIDs(subscriptionID)
		for idx, childID := range childIDs {
			line, found := lo.Find(lines, func(line *billing.Line) bool {
				return lo.FromPtrOr(line.ChildUniqueReferenceID, line.ID) == childID
			})
			s.Truef(found, "line not found with child id %s", childID)
			s.NotNil(line)

			if expectedLine.Qty.IsPresent() {
				if line.UsageBased == nil {
					s.Failf("usage based line not found", "line not found with child id %s", childID)
				} else if line.UsageBased.Quantity == nil {
					s.Failf("usage based line quantity not found", "line not found with child id %s", childID)
				} else {
					s.Equal(expectedLine.Qty.OrEmpty(), line.UsageBased.Quantity.InexactFloat64(), "%s: quantity", childID)
				}
			}

			if expectedLine.Price.IsPresent() {
				s.Equal(*expectedLine.Price.OrEmpty(), *line.UsageBased.Price, "%s: price", childID)
			}

			s.Equal(expectedLine.Periods[idx].Start, line.Period.Start, "%s: period start", childID)
			s.Equal(expectedLine.Periods[idx].End, line.Period.End, "%s: period end", childID)

			if expectedLine.InvoiceAt.IsPresent() {
				s.Equal(expectedLine.InvoiceAt.OrEmpty()[idx], line.InvoiceAt, "%s: invoice at", childID)
			}

			if expectedLine.AdditionalChecks != nil {
				expectedLine.AdditionalChecks(line)
			}
		}
	}
}

type lineMatcher interface {
	ChildIDs(subsID string) []string
}

type recurringLineMatcher struct {
	PhaseKey  string
	ItemKey   string
	Version   int
	PeriodMin int
	PeriodMax int
}

func (m recurringLineMatcher) ChildIDs(subsID string) []string {
	out := []string{}
	for periodID := m.PeriodMin; periodID <= m.PeriodMax; periodID++ {
		out = append(out, fmt.Sprintf("%s/%s/%s/v[%d]/period[%d]", subsID, m.PhaseKey, m.ItemKey, m.Version, periodID))
	}

	return out
}

type oneTimeLineMatcher struct {
	PhaseKey string
	ItemKey  string
	Version  int
}

func (m oneTimeLineMatcher) ChildIDs(subsID string) []string {
	return []string{fmt.Sprintf("%s/%s/%s/v[%d]", subsID, m.PhaseKey, m.ItemKey, m.Version)}
}

// helpers

//nolint:unparam
func (s *SuiteBase) phaseMeta(key string, duration string) productcatalog.PhaseMeta {
	out := productcatalog.PhaseMeta{
		Key:  key,
		Name: key,
	}

	if duration != "" {
		out.Duration = lo.ToPtr(datetime.MustParseDuration(s.T(), duration))
	}

	return out
}

func (s *SuiteBase) enableProgressiveBilling() {
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.ProgressiveBilling = true
	})
}

func (s *SuiteBase) updateProfile(modify func(profile *billing.Profile)) {
	defaultProfile, err := s.BillingService.GetDefaultProfile(s.T().Context(), billing.GetDefaultProfileInput{
		Namespace: s.Namespace,
	})
	s.NoError(err)

	modify(defaultProfile)

	defaultProfile.AppReferences = nil

	_, err = s.BillingService.UpdateProfile(s.T().Context(), billing.UpdateProfileInput(defaultProfile.BaseProfile))
	s.NoError(err)
}

type subscriptionAddItem struct {
	PhaseKey       string
	ItemKey        string
	Price          *productcatalog.Price
	BillingCadence *datetime.ISODuration
	FeatureKey     string
	TaxConfig      *productcatalog.TaxConfig
}

func (i subscriptionAddItem) AsPatch() subscription.Patch {
	var rc productcatalog.RateCard

	meta := productcatalog.RateCardMeta{
		Name:       i.ItemKey,
		Key:        i.ItemKey,
		Price:      i.Price,
		FeatureKey: lo.EmptyableToPtr(i.FeatureKey),
		TaxConfig:  i.TaxConfig,
	}

	switch {
	case i.Price == nil:
		rc = &productcatalog.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: i.BillingCadence,
		}
	case i.Price.Type() == productcatalog.FlatPriceType:
		rc = &productcatalog.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: i.BillingCadence,
		}
	default:
		rc = &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta,
			BillingCadence: *i.BillingCadence,
		}
	}

	return patch.PatchAddItem{
		PhaseKey: i.PhaseKey,
		ItemKey:  i.ItemKey,
		CreateInput: subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: i.PhaseKey,
					ItemKey:  i.ItemKey,
					RateCard: rc,
				},
			},
		},
	}
}

func (s *SuiteBase) generatePeriods(startStr, endStr string, cadenceStr string, n int) []billing.Period { //nolint: unparam
	start := testutils.GetRFC3339Time(s.T(), startStr)
	end := testutils.GetRFC3339Time(s.T(), endStr)
	cadence := datetime.MustParseDuration(s.T(), cadenceStr)

	out := []billing.Period{}

	for n != 0 {
		out = append(out, billing.Period{
			Start: start,
			End:   end,
		})

		start, _ = cadence.AddTo(start)
		end, _ = cadence.AddTo(end)

		n--
	}
	return out
}

// populateChildIDsFromParents copies over the child ID from the parent line, if it's not already set
// as line splitting doesn't set the child ID on child lines to prevent conflicts if multiple split lines
// end up on a single invoice.
func (s *SuiteBase) populateChildIDsFromParents(invoice *billing.Invoice) {
	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID == nil && line.SplitLineGroupID != nil {
			line.ChildUniqueReferenceID = line.SplitLineHierarchy.Group.UniqueReferenceID
		}
	}
}

// helpers

func (s *SuiteBase) createSubscriptionFromPlanPhases(phases []productcatalog.Phase) subscription.SubscriptionView {
	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: phases,
		},
	}

	return s.createSubscriptionFromPlan(planInput)
}

func (s *SuiteBase) createSubscriptionFromPlan(planInput plan.CreatePlanInput) subscription.SubscriptionView {
	ctx := s.T().Context()

	plan, err := s.PlanService.CreatePlan(ctx, planInput)
	s.NoError(err)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(clock.Now()),
			},
			Name: "subs-1",
		},
		Namespace:  s.Namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)
	return subsView
}
