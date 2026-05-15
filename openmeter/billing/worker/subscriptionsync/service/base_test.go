package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
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
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type SuiteBase struct {
	billingtest.BaseSuite
	billingtest.SubscriptionMixin
	Service *Service
	Adapter subscriptionsync.Adapter
	Charges charges.Service

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

func (s *SuiteBase) setupChargesService(config chargestestutils.Config) {
	s.T().Helper()

	stack, err := chargestestutils.NewServices(s.T(), config)
	s.NoError(err)

	s.Charges = stack.ChargesService

	service, err := New(Config{
		BillingService:          s.BillingService,
		ChargesService:          s.Charges,
		Logger:                  slog.Default(),
		Tracer:                  noop.NewTracerProvider().Tracer("test"),
		SubscriptionSyncAdapter: s.Adapter,
		SubscriptionService:     s.SubscriptionService,
	})
	s.NoError(err)

	s.Service = service
}

func (s *SuiteBase) BeforeTest(suiteName, testName string) {
	s.beforeTest(s.T().Context(), suiteName, testName)
}

func (s *SuiteBase) beforeTest(ctx context.Context, suiteName, testName string) {
	s.Namespace = fmt.Sprintf("t-%s-%s-%s", suiteName, testName, ulid.Make().String())

	appSandbox := s.InstallSandboxApp(s.T(), s.Namespace)

	s.ProvisionBillingProfile(ctx, s.Namespace, appSandbox.GetID())

	apiRequestsTotalMeterSlug := "api-requests-total"
	apiRequestsTotalMeterID := ulid.Make().String()

	testMeter := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: apiRequestsTotalMeterID,
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
	}
	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{testMeter})
	s.NoError(err, "Replacing meters must not return error")

	apiRequestsTotalFeatureKey := "api-requests-total"

	apiRequestsTotalFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: s.Namespace,
		Name:      "api-requests-total",
		Key:       apiRequestsTotalFeatureKey,
		MeterID:   lo.ToPtr(apiRequestsTotalMeterID),
	})
	s.NoError(err)
	s.APIRequestsTotalFeature = apiRequestsTotalFeature

	customerEntity := s.CreateTestCustomer(s.Namespace, "test")
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	s.Customer = customerEntity
}

func (s *SuiteBase) AfterTest(suiteName, testName string) {
	s.afterTest(s.T().Context(), suiteName, testName)
}

func (s *SuiteBase) afterTest(ctx context.Context, suiteName, testName string) {
	clock.UnFreeze()
	clock.ResetTime()

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
	s.NoError(err, "Replacing meters must not return error")

	s.MockStreamingConnector.Reset()
	s.Service.featureFlags = FeatureFlags{}
}

func (s *SuiteBase) gatheringInvoice(ctx context.Context, namespace string, customerID string) billing.GatheringInvoice {
	s.T().Helper()

	invoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
			billing.GatheringInvoiceExpandAvailableActions,
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 1, "expected 1 gathering invoice")
	return invoices.Items[0]
}

func (s *SuiteBase) expectNoGatheringInvoice(ctx context.Context, namespace string, customerID string) {
	s.T().Helper()

	invoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.GatheringInvoiceExpands{},
	})

	s.NoError(err)
	if len(invoices.Items) > 0 {
		for _, invoice := range invoices.Items {
			s.DebugDumpInvoice(fmt.Sprintf("unexpected gathering invoice[%s]", invoice.ID), invoice)
		}
	}
	s.Len(invoices.Items, 0)
}

func (s *SuiteBase) enableProrating() {
	s.Service.featureFlags.EnableFlatFeeInAdvanceProrating = true
	s.Service.featureFlags.EnableFlatFeeInArrearsProrating = true
}

func (s *SuiteBase) getGatheringLineByChildID(invoice billing.GatheringInvoice, childID string) *billing.GatheringLine {
	s.T().Helper()

	for idx, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			return &invoice.Lines.OrEmpty()[idx]
		}
	}

	s.Failf("line not found", "line with child id %s not found", childID)

	return nil
}

func (s *SuiteBase) getStandardLineByChildID(invoice billing.StandardInvoice, childID string) *billing.StandardLine {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			return line
		}
	}

	s.Failf("line not found", "line with child id %s not found", childID)

	return nil
}

func (s *SuiteBase) expectNoLineWithChildID(invoice billing.GenericInvoiceReader, childID string) {
	s.T().Helper()

	for _, line := range invoice.GetGenericLines().OrEmpty() {
		if line.GetChildUniqueReferenceID() != nil && *line.GetChildUniqueReferenceID() == childID {
			s.Failf("line found", "line with child id %s found", childID)
		}
	}
}

func (s *SuiteBase) timingImmediate() subscription.Timing {
	return subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	}
}

func (s *SuiteBase) mustParseTime(t string) time.Time {
	s.T().Helper()
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *SuiteBase) testContext() context.Context {
	s.T().Helper()
	return s.T().Context()
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
	Periods          []timeutil.ClosedPeriod
	InvoiceAt        mo.Option[[]time.Time]
	Charge           mo.Option[chargeExpects]
	AdditionalChecks func(line billing.GenericInvoiceLine)
}

type chargeExpects struct {
	Status         string
	SettlementMode productcatalog.SettlementMode
}

func (s *SuiteBase) expectLines(invoice billing.GenericInvoiceReader, subscriptionID string, expectedLines []expectedLine) {
	s.T().Helper()

	lines := invoice.GetGenericLines()
	if lines.IsAbsent() {
		s.Failf("lines not found", "lines not found for invoice %s", invoice.GetID())
	}

	existingLineChildIDs := lo.Map(lines.OrEmpty(), func(line billing.GenericInvoiceLine, _ int) string {
		return lo.FromPtrOr(line.GetChildUniqueReferenceID(), line.GetID())
	})

	expectedLineIDs := lo.Flatten(lo.Map(expectedLines, func(expectedLine expectedLine, _ int) []string {
		return expectedLine.Matcher.ChildIDs(subscriptionID)
	}))

	s.ElementsMatch(expectedLineIDs, existingLineChildIDs)

	for _, expectedLine := range expectedLines {
		childIDs := expectedLine.Matcher.ChildIDs(subscriptionID)
		for idx, childID := range childIDs {
			line, found := lo.Find(lines.OrEmpty(), func(line billing.GenericInvoiceLine) bool {
				return lo.FromPtrOr(line.GetChildUniqueReferenceID(), line.GetID()) == childID
			})
			s.Truef(found, "line not found with child id %s", childID)
			s.NotNil(line)

			if expectedLine.Qty.IsPresent() {
				lineQuantityAccessor, ok := line.(billing.QuantityAccessor)
				if !ok {
					s.Failf("line is not a quantity accessor", "line is not a quantity accessor with child id %s", childID)
				}

				lineQuantity := lineQuantityAccessor.GetQuantity()
				if lineQuantity == nil {
					s.Failf("line quantity not found", "line quantity not found with child id %s", childID)
				} else {
					s.Equal(expectedLine.Qty.OrEmpty(), lineQuantity.InexactFloat64(), "%s: quantity", childID)
				}
			}

			if expectedLine.Price.IsPresent() {
				s.Equal(*expectedLine.Price.OrEmpty(), *line.GetPrice(), "%s: price", childID)
			}

			s.Equal(expectedLine.Periods[idx].From, line.GetServicePeriod().From, "%s: period start", childID)
			s.Equal(expectedLine.Periods[idx].To, line.GetServicePeriod().To, "%s: period end", childID)

			if expectedLine.InvoiceAt.IsPresent() {
				invoiceAtAccessor, ok := line.(billing.InvoiceAtAccessor)
				if !ok {
					s.Failf("line is not a invoice at accessor", "line is not a invoice at accessor with child id %s", childID)
				}

				invoiceAt := invoiceAtAccessor.GetInvoiceAt()
				s.Equal(expectedLine.InvoiceAt.OrEmpty()[idx], invoiceAt, "%s: invoice at", childID)
			}

			if expectedLine.AdditionalChecks != nil {
				expectedLine.AdditionalChecks(line)
			}
		}
	}
}

func (s *SuiteBase) assertChargesForLines(ctx context.Context, subsView subscription.SubscriptionView, expectedLines []expectedLine) {
	s.T().Helper()

	flatFeeExpectedLines := make([]expectedLine, 0, len(expectedLines))
	usageBasedExpectedLines := make([]expectedLine, 0, len(expectedLines))

	for _, expectedLine := range expectedLines {
		s.Require().True(expectedLine.Price.IsPresent())

		switch expectedLine.Price.OrEmpty().Type() {
		case productcatalog.FlatPriceType:
			flatFeeExpectedLines = append(flatFeeExpectedLines, expectedLine)
		case productcatalog.UnitPriceType:
			usageBasedExpectedLines = append(usageBasedExpectedLines, expectedLine)
		default:
			s.Failf("unsupported charge price", "unsupported charge price type %s", expectedLine.Price.OrEmpty().Type())
		}
	}

	s.assertFlatFeeChargesForLines(ctx, subsView, flatFeeExpectedLines)
	s.assertUsageBasedChargesForLines(ctx, subsView, usageBasedExpectedLines)
}

func (s *SuiteBase) assertFlatFeeChargesForLines(ctx context.Context, subsView subscription.SubscriptionView, expectedLines []expectedLine) {
	s.T().Helper()

	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subsView.Subscription.ID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeFlatFee},
	})
	s.NoError(err)

	flatFeeCharges := make([]flatfee.Charge, 0, len(res.Items))
	for _, charge := range res.Items {
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		flatFeeCharges = append(flatFeeCharges, flatFeeCharge)
	}

	expectedChargeIDs := lo.Flatten(lo.Map(expectedLines, func(expectedLine expectedLine, _ int) []string {
		return expectedLine.Matcher.ChildIDs(subsView.Subscription.ID)
	}))
	actualChargeIDs := lo.Map(flatFeeCharges, func(charge flatfee.Charge, _ int) string {
		s.Require().NotNil(charge.Intent.UniqueReferenceID)
		return lo.FromPtr(charge.Intent.UniqueReferenceID)
	})
	s.Len(flatFeeCharges, len(expectedChargeIDs))
	s.ElementsMatch(expectedChargeIDs, actualChargeIDs)

	expectedItemVersionCounts := map[string]int{}
	for _, expectedLine := range expectedLines {
		phaseKey, itemKey, version := s.chargeExpectedLineMatcherParts(expectedLine.Matcher)

		key := fmt.Sprintf("%s/%s", phaseKey, itemKey)
		expectedItemVersionCounts[key] = max(expectedItemVersionCounts[key], version+1)
	}

	for key, expectedVersionCount := range expectedItemVersionCounts {
		parts := strings.Split(key, "/")
		s.Require().Len(parts, 2)

		phase := s.getPhaseByKey(s.T(), subsView, parts[0])
		s.Require().Len(phase.ItemsByKey[parts[1]], expectedVersionCount)
	}

	for _, expectedLine := range expectedLines {
		phaseKey, itemKey, version := s.chargeExpectedLineMatcherParts(expectedLine.Matcher)
		phase := s.getPhaseByKey(s.T(), subsView, phaseKey)
		item := phase.ItemsByKey[itemKey][version]

		s.Require().True(expectedLine.Price.IsPresent())
		flatPrice, err := expectedLine.Price.OrEmpty().AsFlat()
		s.NoError(err)
		catalogFlatPrice, err := item.Spec.RateCard.AsMeta().Price.AsFlat()
		s.NoError(err)
		s.Require().True(expectedLine.Charge.IsPresent(), "charge expectations are required")
		chargeExpects := expectedLine.Charge.OrEmpty()
		s.Require().NotEmpty(chargeExpects.Status, "charge status expectation is required")
		s.Require().NotEmpty(chargeExpects.SettlementMode, "charge settlement mode expectation is required")

		for idx, childID := range expectedLine.Matcher.ChildIDs(subsView.Subscription.ID) {
			charge, found := lo.Find(flatFeeCharges, func(charge flatfee.Charge) bool {
				return charge.Intent.UniqueReferenceID != nil && *charge.Intent.UniqueReferenceID == childID
			})
			s.Require().Truef(found, "flat fee charge not found with child unique reference id %s", childID)

			s.Equal(chargeExpects.Status, string(charge.Status), "%s: status", childID)
			s.Equal(chargeExpects.SettlementMode, charge.Intent.SettlementMode, "%s: settlement mode", childID)
			s.Equal(expectedLine.Periods[idx], charge.Intent.ServicePeriod, "%s: service period", childID)
			if expectedLine.InvoiceAt.IsPresent() {
				s.Equal(expectedLine.InvoiceAt.OrEmpty()[idx], charge.Intent.InvoiceAt, "%s: invoice at", childID)
			}
			s.Equal(s.Customer.ID, charge.Intent.CustomerID, "%s: customer id", childID)
			s.Equal(string(currency.USD), string(charge.Intent.Currency), "%s: currency", childID)
			s.Equal(flatPrice.PaymentTerm, charge.Intent.PaymentTerm, "%s: payment term", childID)
			s.AssertDecimalEqual(catalogFlatPrice.Amount, charge.Intent.AmountBeforeProration, fmt.Sprintf("%s: amount before proration", childID))
			s.AssertDecimalEqual(flatPrice.Amount, charge.State.AmountAfterProration, fmt.Sprintf("%s: amount after proration", childID))
			s.Require().NotNil(charge.Intent.Subscription, "%s: subscription", childID)
			s.Equal(subsView.Subscription.ID, charge.Intent.Subscription.SubscriptionID, "%s: subscription id", childID)
			s.Equal(phase.SubscriptionPhase.ID, charge.Intent.Subscription.PhaseID, "%s: phase id", childID)
			s.Equal(item.SubscriptionItem.ID, charge.Intent.Subscription.ItemID, "%s: item id", childID)
		}
	}
}

func (s *SuiteBase) assertUsageBasedChargesForLines(ctx context.Context, subsView subscription.SubscriptionView, expectedLines []expectedLine) {
	s.T().Helper()

	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subsView.Subscription.ID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeUsageBased},
	})
	s.NoError(err)

	usageBasedCharges := make([]usagebased.Charge, 0, len(res.Items))
	for _, charge := range res.Items {
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)
		usageBasedCharges = append(usageBasedCharges, usageBasedCharge)
	}

	expectedChargeIDs := lo.Flatten(lo.Map(expectedLines, func(expectedLine expectedLine, _ int) []string {
		return expectedLine.Matcher.ChildIDs(subsView.Subscription.ID)
	}))
	actualChargeIDs := lo.Map(usageBasedCharges, func(charge usagebased.Charge, _ int) string {
		s.Require().NotNil(charge.Intent.UniqueReferenceID)
		return lo.FromPtr(charge.Intent.UniqueReferenceID)
	})
	s.Len(usageBasedCharges, len(expectedChargeIDs))
	s.ElementsMatch(expectedChargeIDs, actualChargeIDs)

	expectedItemVersionCounts := map[string]int{}
	for _, expectedLine := range expectedLines {
		phaseKey, itemKey, version := s.chargeExpectedLineMatcherParts(expectedLine.Matcher)

		key := fmt.Sprintf("%s/%s", phaseKey, itemKey)
		expectedItemVersionCounts[key] = max(expectedItemVersionCounts[key], version+1)
	}

	for key, expectedVersionCount := range expectedItemVersionCounts {
		parts := strings.Split(key, "/")
		s.Require().Len(parts, 2)

		phase := s.getPhaseByKey(s.T(), subsView, parts[0])
		s.Require().Len(phase.ItemsByKey[parts[1]], expectedVersionCount)
	}

	for _, expectedLine := range expectedLines {
		phaseKey, itemKey, version := s.chargeExpectedLineMatcherParts(expectedLine.Matcher)
		phase := s.getPhaseByKey(s.T(), subsView, phaseKey)
		item := phase.ItemsByKey[itemKey][version]

		s.Require().True(expectedLine.Price.IsPresent())
		unitPrice, err := expectedLine.Price.OrEmpty().AsUnit()
		s.NoError(err)
		s.Require().True(expectedLine.Charge.IsPresent(), "charge expectations are required")
		chargeExpects := expectedLine.Charge.OrEmpty()
		s.Require().NotEmpty(chargeExpects.Status, "charge status expectation is required")
		s.Require().NotEmpty(chargeExpects.SettlementMode, "charge settlement mode expectation is required")

		for idx, childID := range expectedLine.Matcher.ChildIDs(subsView.Subscription.ID) {
			charge, found := lo.Find(usageBasedCharges, func(charge usagebased.Charge) bool {
				return charge.Intent.UniqueReferenceID != nil && *charge.Intent.UniqueReferenceID == childID
			})
			s.Require().Truef(found, "usage-based charge not found with child unique reference id %s", childID)

			s.Equal(chargeExpects.Status, string(charge.Status), "%s: status", childID)
			s.Equal(chargeExpects.SettlementMode, charge.Intent.SettlementMode, "%s: settlement mode", childID)
			s.Equal(expectedLine.Periods[idx], charge.Intent.ServicePeriod, "%s: service period", childID)
			if expectedLine.InvoiceAt.IsPresent() {
				s.Equal(expectedLine.InvoiceAt.OrEmpty()[idx], charge.Intent.InvoiceAt, "%s: invoice at", childID)
			}
			s.Equal(s.Customer.ID, charge.Intent.CustomerID, "%s: customer id", childID)
			expectedFeatureKey := lo.FromPtrOr(item.Spec.RateCard.AsMeta().FeatureKey, itemKey)
			s.Equal(expectedFeatureKey, charge.Intent.FeatureKey, "%s: feature key", childID)
			s.Equal(*productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: unitPrice.Amount}), charge.Intent.Price, "%s: price", childID)
			s.Require().NotNil(charge.Intent.Subscription, "%s: subscription", childID)
			s.Equal(subsView.Subscription.ID, charge.Intent.Subscription.SubscriptionID, "%s: subscription id", childID)
			s.Equal(phase.SubscriptionPhase.ID, charge.Intent.Subscription.PhaseID, "%s: phase id", childID)
			s.Equal(item.SubscriptionItem.ID, charge.Intent.Subscription.ItemID, "%s: item id", childID)
			s.Nil(charge.State.CurrentRealizationRunID, "%s: current realization run", childID)
			s.Nil(charge.State.AdvanceAfter, "%s: advance after", childID)
		}
	}
}

func (s *SuiteBase) chargeExpectedLineMatcherParts(matcher lineMatcher) (string, string, int) {
	s.T().Helper()

	switch matcher := matcher.(type) {
	case recurringLineMatcher:
		return matcher.PhaseKey, matcher.ItemKey, matcher.Version
	case oneTimeLineMatcher:
		return matcher.PhaseKey, matcher.ItemKey, matcher.Version
	default:
		s.T().Fatalf("charge assertion does not support matcher type %T", matcher)
		return "", "", 0
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

func (s *SuiteBase) generatePeriods(startStr, endStr string, cadenceStr string, n int) []timeutil.ClosedPeriod { //nolint: unparam
	start := testutils.GetRFC3339Time(s.T(), startStr)
	end := testutils.GetRFC3339Time(s.T(), endStr)
	cadence := datetime.MustParseDuration(s.T(), cadenceStr)

	out := []timeutil.ClosedPeriod{}

	for n != 0 {
		out = append(out, timeutil.ClosedPeriod{
			From: start,
			To:   end,
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
func (s *SuiteBase) populateChildIDsFromParents(invoice billing.GenericInvoice) {
	genericLinesOption := invoice.GetGenericLines()
	if genericLinesOption.IsAbsent() {
		s.Failf("lines not found", "lines not found for invoice %s", invoice.GetID())
	}

	genericLines := genericLinesOption.OrEmpty()

	for idx, line := range genericLines {
		if line.GetChildUniqueReferenceID() == nil && line.GetSplitLineGroupID() != nil {
			invoiceLine := line.AsInvoiceLine()
			switch invoiceLine.Type() {
			case billing.InvoiceLineTypeStandard:
				stdInvoiceLine, err := invoiceLine.AsStandardLine()
				s.NoError(err)

				line.SetChildUniqueReferenceID(stdInvoiceLine.SplitLineHierarchy.Group.UniqueReferenceID)
			case billing.InvoiceLineTypeGathering:
				splitLineGroupID := line.GetSplitLineGroupID()
				if splitLineGroupID == nil {
					s.Failf("split line group id not found", "split line group id not found for line %s", line.GetID())
					return
				}

				splitLineGroup, err := s.BillingAdapter.GetSplitLineGroup(s.T().Context(), billing.GetSplitLineGroupInput{
					Namespace: s.Namespace,
					ID:        *splitLineGroupID,
				})
				s.NoError(err)

				line.SetChildUniqueReferenceID(splitLineGroup.Group.UniqueReferenceID)
			default:
				s.Failf("unexpected line type", "unexpected line type %s for line %s", invoiceLine.Type(), line.GetID())
			}
		}

		genericLines[idx] = line
	}

	err := invoice.SetLines(genericLines)
	s.NoError(err)
}

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
	return s.createSubscriptionFromPlanAt(planInput, clock.Now())
}

func (s *SuiteBase) createSubscriptionFromPlanAt(planInput plan.CreatePlanInput, startAt time.Time) subscription.SubscriptionView {
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
				Custom: lo.ToPtr(startAt),
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
