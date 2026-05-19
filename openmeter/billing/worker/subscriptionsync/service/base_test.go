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
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
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
				expectedPrice := expectedLine.Price.OrEmpty()
				actualPrice := line.GetPrice()
				s.Truef(expectedPrice.Equal(actualPrice), "%s: price expected %v, got %v", childID, expectedPrice, actualPrice)
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

type expectedCharge struct {
	Matcher            lineMatcher
	Type               chargesmeta.ChargeType
	Status             string
	Price              *productcatalog.Price
	Periods            []timeutil.ClosedPeriod
	FullServicePeriods []timeutil.ClosedPeriod
	BillingPeriods     []timeutil.ClosedPeriod
	InvoiceAt          []*time.Time
	GatheringLines     []expectedChargeGatheringLine
	Realizations       []expectedChargeRealization
}

type expectedChargeGatheringLine struct {
	LineMatcher lineMatcher
	Period      timeutil.ClosedPeriod
	Price       *productcatalog.Price
	InvoiceAt   *time.Time
}

type expectedChargeRealization struct {
	LineMatcher lineMatcher
	Period      timeutil.ClosedPeriod
	Status      billing.StandardInvoiceStatus
	IsVoided    bool
	Price       *productcatalog.Price
	Totals      totals.Totals
}

type actualChargeGatheringLine struct {
	Period    timeutil.ClosedPeriod
	Price     *productcatalog.Price
	InvoiceAt time.Time
}

type chargeRealizationKey struct {
	Period   timeutil.ClosedPeriod
	Status   billing.StandardInvoiceStatus
	IsVoided bool
}

func (s *SuiteBase) assertCharges(ctx context.Context, subscriptionID models.NamespacedID, expectedCharges []expectedCharge) {
	s.T().Helper()

	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       subscriptionID.Namespace,
		SubscriptionIDs: []string{subscriptionID.ID},
		IncludeDeleted:  true,
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		},
	})
	s.NoError(err)

	expectedChargeIDs := lo.Flatten(lo.Map(expectedCharges, func(expectedCharge expectedCharge, _ int) []string {
		return expectedCharge.Matcher.ChildIDs(subscriptionID.ID)
	}))
	actualChargeIDs := lo.Map(res.Items, func(charge charges.Charge, _ int) string {
		uniqueReferenceID, err := charge.GetUniqueReferenceID()
		s.NoError(err)
		s.Require().NotNil(uniqueReferenceID, "charge %s child unique reference id", charge.GetID())

		return *uniqueReferenceID
	})

	s.Len(res.Items, len(expectedChargeIDs))
	s.ElementsMatch(expectedChargeIDs, actualChargeIDs)

	for _, expectedCharge := range expectedCharges {
		childIDs := expectedCharge.Matcher.ChildIDs(subscriptionID.ID)
		s.Require().NotEmpty(expectedCharge.Type, "expected charge type")
		s.Require().NotEmpty(expectedCharge.Status, "expected charge status")
		s.Require().Len(expectedCharge.Periods, len(childIDs), "expected charge periods")
		if len(expectedCharge.InvoiceAt) > 0 {
			s.Require().Len(expectedCharge.InvoiceAt, len(childIDs), "expected charge invoice at")
		}
		if len(expectedCharge.FullServicePeriods) > 0 {
			s.Require().Len(expectedCharge.FullServicePeriods, len(childIDs), "expected charge full service periods")
		}
		if len(expectedCharge.BillingPeriods) > 0 {
			s.Require().Len(expectedCharge.BillingPeriods, len(childIDs), "expected charge billing periods")
		}

		for idx, childID := range childIDs {
			charge, found := lo.Find(res.Items, func(charge charges.Charge) bool {
				uniqueReferenceID, err := charge.GetUniqueReferenceID()
				s.NoError(err)

				return uniqueReferenceID != nil && *uniqueReferenceID == childID
			})
			s.Require().Truef(found, "charge not found with child unique reference id %s", childID)

			s.assertCharge(ctx, charge, subscriptionID.ID, childID, childIDs, expectedCharge, idx)
		}
	}
}

func (s *SuiteBase) assertCharge(ctx context.Context, charge charges.Charge, subscriptionID string, childID string, expectedChargeIDs []string, expectedCharge expectedCharge, idx int) {
	s.T().Helper()

	chargeID, err := charge.GetChargeID()
	s.NoError(err)

	s.Equal(expectedCharge.Type, charge.Type(), "%s: type", childID)

	switch charge.Type() {
	case chargesmeta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)

		s.Equal(expectedCharge.Status, string(usageBasedCharge.Status), "%s: status", childID)
		s.Equal(expectedCharge.Periods[idx], usageBasedCharge.Intent.ServicePeriod, "%s: service period", childID)
		if len(expectedCharge.FullServicePeriods) > 0 {
			s.Equal(expectedCharge.FullServicePeriods[idx], usageBasedCharge.Intent.FullServicePeriod, "%s: full service period", childID)
		}
		if len(expectedCharge.BillingPeriods) > 0 {
			s.Equal(expectedCharge.BillingPeriods[idx], usageBasedCharge.Intent.BillingPeriod, "%s: billing period", childID)
		}
		if expectedCharge.Price != nil {
			s.Truef(expectedCharge.Price.Equal(&usageBasedCharge.Intent.Price), "%s: price expected %v, got %v", childID, expectedCharge.Price, usageBasedCharge.Intent.Price)
		}
		if len(expectedCharge.InvoiceAt) > idx && expectedCharge.InvoiceAt[idx] != nil {
			s.Equal(*expectedCharge.InvoiceAt[idx], usageBasedCharge.Intent.InvoiceAt, "%s: invoice at", childID)
		}
	case chargesmeta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(expectedCharge.Status, string(flatFeeCharge.Status), "%s: status", childID)
		s.Equal(expectedCharge.Periods[idx], flatFeeCharge.Intent.ServicePeriod, "%s: service period", childID)
		if len(expectedCharge.FullServicePeriods) > 0 {
			s.Equal(expectedCharge.FullServicePeriods[idx], flatFeeCharge.Intent.FullServicePeriod, "%s: full service period", childID)
		}
		if len(expectedCharge.BillingPeriods) > 0 {
			s.Equal(expectedCharge.BillingPeriods[idx], flatFeeCharge.Intent.BillingPeriod, "%s: billing period", childID)
		}
		if expectedCharge.Price != nil {
			expectedFlatPrice, err := expectedCharge.Price.AsFlat()
			s.NoError(err)
			s.AssertDecimalEqual(expectedFlatPrice.Amount, flatFeeCharge.Intent.AmountBeforeProration, fmt.Sprintf("%s: amount before proration", childID))
		}
		if len(expectedCharge.InvoiceAt) > idx && expectedCharge.InvoiceAt[idx] != nil {
			s.Equal(*expectedCharge.InvoiceAt[idx], flatFeeCharge.Intent.InvoiceAt, "%s: invoice at", childID)
		}
	default:
		s.Failf("unsupported charge type", "charge %s has unsupported type %s", chargeID.ID, charge.Type())
	}

	s.assertChargeGatheringLines(ctx, charge, subscriptionID, childID, expectedChargeIDs, expectedCharge.Periods[idx], expectedCharge.Price, expectedCharge.GatheringLines)

	expectedRealizations := s.expectedRealizationsForCharge(subscriptionID, childID, expectedChargeIDs, expectedCharge.Periods[idx], expectedCharge.Realizations)
	actualRealizations := s.chargeRealizations(ctx, charge)
	expectedRealizationKeys := lo.Map(expectedRealizations, func(realization expectedChargeRealization, _ int) chargeRealizationKey {
		return chargeRealizationKey{
			Period:   realization.Period,
			Status:   realization.Status,
			IsVoided: realization.IsVoided,
		}
	})
	actualRealizationKeys := lo.Map(actualRealizations, func(realization actualChargeRealization, _ int) chargeRealizationKey {
		return chargeRealizationKey{
			Period:   realization.Period,
			Status:   realization.Status,
			IsVoided: realization.IsVoided,
		}
	})

	s.ElementsMatch(expectedRealizationKeys, actualRealizationKeys, "%s: realizations", childID)

	for _, expectedRealization := range expectedRealizations {
		actualRealization, found := lo.Find(actualRealizations, func(realization actualChargeRealization) bool {
			return realization.Period == expectedRealization.Period &&
				realization.Status == expectedRealization.Status &&
				realization.IsVoided == expectedRealization.IsVoided
		})
		if !found {
			s.Failf("realization not found", "realization not found for charge %s with status %s and period %s", childID, expectedRealization.Status, expectedRealization.Period)
			continue
		}

		expectedPrice := expectedRealization.Price
		if expectedPrice == nil {
			expectedPrice = expectedCharge.Price
		}
		if expectedPrice != nil {
			s.Truef(expectedPrice.Equal(actualRealization.Price), "%s: realization price expected %v, got %v", childID, expectedPrice, actualRealization.Price)
		}

		if !expectedRealization.Totals.IsZero() {
			s.Truef(expectedRealization.Totals.Equal(actualRealization.Totals), "%s: realization totals expected %v, got %v", childID, expectedRealization.Totals, actualRealization.Totals)
		}

	}
}

func (s *SuiteBase) assertChargeGatheringLines(ctx context.Context, charge charges.Charge, subscriptionID string, childID string, expectedChargeIDs []string, chargePeriod timeutil.ClosedPeriod, chargePrice *productcatalog.Price, expectedGatheringLines []expectedChargeGatheringLine) {
	s.T().Helper()

	expectedLines := s.expectedGatheringLinesForCharge(subscriptionID, childID, expectedChargeIDs, chargePeriod, expectedGatheringLines)
	actualLines := s.gatheringChargeLines(ctx, charge)

	expectedKeys := lo.Map(expectedLines, func(line expectedChargeGatheringLine, _ int) timeutil.ClosedPeriod {
		return line.Period
	})
	actualKeys := lo.Map(actualLines, func(line actualChargeGatheringLine, _ int) timeutil.ClosedPeriod {
		return line.Period
	})

	s.ElementsMatch(expectedKeys, actualKeys, "%s: gathering lines", childID)

	for _, expectedLine := range expectedLines {
		actualLine, found := lo.Find(actualLines, func(line actualChargeGatheringLine) bool {
			return line.Period == expectedLine.Period
		})
		if !found {
			s.Failf("gathering line not found", "gathering line not found for charge %s with period %s", childID, expectedLine.Period)
			continue
		}

		expectedPrice := expectedLine.Price
		if expectedPrice == nil {
			expectedPrice = chargePrice
		}
		if expectedPrice != nil {
			s.Truef(expectedPrice.Equal(actualLine.Price), "%s: gathering line price expected %v, got %v", childID, expectedPrice, actualLine.Price)
		}
		if expectedLine.InvoiceAt != nil {
			s.Equal(*expectedLine.InvoiceAt, actualLine.InvoiceAt, "%s: gathering line invoice at", childID)
		}
	}
}

func (s *SuiteBase) expectedGatheringLinesForCharge(subscriptionID string, childID string, expectedChargeIDs []string, chargePeriod timeutil.ClosedPeriod, expectedGatheringLines []expectedChargeGatheringLine) []expectedChargeGatheringLine {
	s.T().Helper()

	return lo.FilterMap(expectedGatheringLines, func(line expectedChargeGatheringLine, _ int) (expectedChargeGatheringLine, bool) {
		if line.LineMatcher == nil {
			s.Require().Len(expectedChargeIDs, 1, "%s: gathering line matcher is required when a charge expectation expands to multiple charges", childID)
			if lo.IsEmpty(line.Period) {
				line.Period = chargePeriod
			}

			return line, true
		}

		gatheringLineChildIDs := line.LineMatcher.ChildIDs(subscriptionID)
		matchingChargeIDs := lo.Intersect(gatheringLineChildIDs, expectedChargeIDs)
		s.Require().NotEmpty(matchingChargeIDs, "%s: gathering line matcher must belong to the charge expectation", childID)

		if len(expectedChargeIDs) == 1 {
			s.Require().Contains(gatheringLineChildIDs, childID, "%s: gathering line matcher must match the charge", childID)
		}

		if lo.IsEmpty(line.Period) {
			line.Period = chargePeriod
		}

		return line, lo.Contains(gatheringLineChildIDs, childID)
	})
}

func (s *SuiteBase) expectedRealizationsForCharge(subscriptionID string, childID string, expectedChargeIDs []string, chargePeriod timeutil.ClosedPeriod, expectedRealizations []expectedChargeRealization) []expectedChargeRealization {
	s.T().Helper()

	return lo.FilterMap(expectedRealizations, func(realization expectedChargeRealization, _ int) (expectedChargeRealization, bool) {
		if realization.LineMatcher == nil {
			s.Require().Len(expectedChargeIDs, 1, "%s: realization matcher is required when a charge expectation expands to multiple charges", childID)
			if lo.IsEmpty(realization.Period) {
				realization.Period = chargePeriod
			}

			return realization, true
		}

		realizationChildIDs := realization.LineMatcher.ChildIDs(subscriptionID)
		matchingChargeIDs := lo.Intersect(realizationChildIDs, expectedChargeIDs)
		s.Require().NotEmpty(matchingChargeIDs, "%s: realization matcher must belong to the charge expectation", childID)

		if len(expectedChargeIDs) == 1 {
			s.Require().Contains(realizationChildIDs, childID, "%s: realization matcher must match the charge", childID)
		}

		if lo.IsEmpty(realization.Period) {
			realization.Period = chargePeriod
		}

		return realization, lo.Contains(realizationChildIDs, childID)
	})
}

func (s *SuiteBase) chargeRealizations(ctx context.Context, charge charges.Charge) []actualChargeRealization {
	s.T().Helper()

	chargeID, err := charge.GetChargeID()
	s.NoError(err)

	var out []actualChargeRealization

	switch charge.Type() {
	case chargesmeta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)

		for _, run := range usageBasedCharge.Realizations {
			if run.DeletedAt != nil {
				continue
			}
			if run.InvoiceID == nil || run.LineID == nil {
				continue
			}

			realization := s.standardLineChargeRealization(ctx, billing.InvoiceID{
				Namespace: chargeID.Namespace,
				ID:        *run.InvoiceID,
			}, *run.LineID)
			realization.IsVoided = run.IsVoidedBillingHistory()

			out = append(out, realization)
		}
	case chargesmeta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		runs := flatFeeCharge.Realizations.PriorRuns
		if flatFeeCharge.Realizations.CurrentRun != nil {
			runs = append(runs, *flatFeeCharge.Realizations.CurrentRun)
		}

		for _, run := range runs {
			if run.DeletedAt != nil {
				continue
			}
			if run.InvoiceID == nil || run.LineID == nil {
				continue
			}

			realization := s.standardLineChargeRealization(ctx, billing.InvoiceID{
				Namespace: chargeID.Namespace,
				ID:        *run.InvoiceID,
			}, *run.LineID)
			realization.IsVoided = run.IsVoidedBillingHistory()

			out = append(out, realization)
		}
	}

	return out
}

type actualChargeRealization struct {
	Period   timeutil.ClosedPeriod
	Status   billing.StandardInvoiceStatus
	IsVoided bool
	Price    *productcatalog.Price
	Totals   totals.Totals
}

func (s *SuiteBase) standardLineChargeRealization(ctx context.Context, invoiceID billing.InvoiceID, lineID string) actualChargeRealization {
	s.T().Helper()

	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)

	line := invoice.Lines.GetByID(lineID)
	s.Require().NotNil(line, "standard invoice line %s", lineID)

	return actualChargeRealization{
		Period: line.Period,
		Status: invoice.Status,
		Price:  line.GetPrice(),
		Totals: line.Totals,
	}
}

func (s *SuiteBase) gatheringChargeLines(ctx context.Context, charge charges.Charge) []actualChargeGatheringLine {
	s.T().Helper()

	chargeID, err := charge.GetChargeID()
	s.NoError(err)

	customerID, err := charge.GetCustomerID()
	s.NoError(err)

	invoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{chargeID.Namespace},
		Customers:  []string{customerID.ID},
		Page: pagination.Page{
			PageSize:   100,
			PageNumber: 1,
		},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
		},
	})
	s.NoError(err)

	var out []actualChargeGatheringLine
	for _, invoice := range invoices.Items {
		for _, line := range invoice.Lines.OrEmpty() {
			if line.ChargeID == nil || *line.ChargeID != chargeID.ID {
				continue
			}

			out = append(out, actualChargeGatheringLine{
				Period:    line.ServicePeriod,
				Price:     line.GetPrice(),
				InvoiceAt: line.GetInvoiceAt(),
			})
		}
	}

	return out
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
