package rte_test

import (
	"bytes"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/upstream"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

func (s *RTEIntegrationSuite) TestUpstreamUsageTargetsActiveDemand() {
	// given: a real metered entitlement and credit-only charge share a meter,
	// while tracked balances cover several currencies, feature scopes and owners.
	ctx := s.T().Context()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	ns := s.GetUniqueNamespace("rte-upstream")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID())
	cust := s.CreateLedgerBackedCustomer(ns, "rte-upstream-customer")
	other := s.CreateLedgerBackedCustomer(ns, "other-customer")
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
	ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
		CustomerID: cust.GetID(),
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID: &feature.Feature.ID, EntitlementType: entitlement.EntitlementTypeMetered,
			IssueAfterReset: lo.ToPtr(float64(100)),
			UsagePeriod:     lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{Anchor: start, Interval: timeutil.RecurrencePeriodMonth})),
		},
	})
	s.Require().NoError(err)
	_, err = s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(
		s.CreateMockChargeIntent(creditstest.CreateMockChargeIntentInput{
			Customer: cust.GetID(), Currency: creditstest.USD,
			ServicePeriod:  timeutil.ClosedPeriod{From: start, To: start.AddDate(0, 1, 0)},
			SettlementMode: productcatalog.CreditOnlySettlementMode,
			Price:          productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			Name:           "Usage", ManagedBy: billing.SubscriptionManagedLine,
			UniqueReferenceID: "rte-upstream-usage", FeatureKey: feature.Feature.Key,
		}),
	)})
	s.Require().NoError(err)
	logger := testutils.NewDiscardLogger(s.T())
	repository, err := rteadapter.NewDemandRepository(rteadapter.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	resolver, err := upstream.New(upstream.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)

	all := creditbalance.Key{CustomerID: cust.GetID(), Currency: currencies.NewCurrencyReference("USD")}
	featureKey := all
	featureKey.FeatureFilter = customerbalance.NewFeatureFilter([]string{feature.Feature.Key})
	unrestricted := all
	unrestricted.FeatureFilter = customerbalance.NewUnrestrictedFeatureFilter()
	otherFeature := all
	otherFeature.FeatureFilter = customerbalance.NewFeatureFilter([]string{"another-feature"})
	otherCurrency := all
	otherCurrency.Currency = currencies.NewCurrencyReference("EUR")
	otherCustomer := all
	otherCustomer.CustomerID = other.GetID()
	meteredKey := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}
	for _, key := range []rte.Key{all, featureKey, unrestricted, otherFeature, otherCurrency, otherCustomer, meteredKey} {
		target, err := rte.NewCalculationRequestedEvent(key)
		s.Require().NoError(err)
		// JSON decoding accepts this spelling. The SQL selectors must still
		// distinguish currency and feature scope after registration.
		target.Key = bytes.Replace(target.Key, []byte(`"currency":`), []byte(`"Currency":`), 1)
		s.Require().NoError(repository.Activate(ctx, rte.ActivateDemandInput{Target: target.CalculationTarget}))
		defer func() {
			s.Require().NoError(repository.Deactivate(ctx, target.CalculationTarget))
		}()
	}

	now := start.Add(48 * time.Hour)
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	event := ingestevents.EventBatchedIngest{
		Namespace: eventmodels.NamespaceID{ID: ns}, SubjectKey: "rte-upstream-customer",
		MeterSlugs: []string{"api-requests-total", "api-requests-total"}, StoredAt: now,
		RawEvents: []serializer.CloudEventsKafkaPayload{{Time: start.Add(time.Hour).Unix()}, {Time: start.Add(time.Hour).Unix()}},
	}

	// when: late duplicate usage is resolved after arrival, only the matching
	// active demand identities are selected once each.
	selectors, err := resolver.Resolve(ctx, &event)
	s.Require().NoError(err)
	hashes := func(selectors []rte.DemandSelector) []string {
		var result []string
		for _, selector := range selectors {
			for _, target := range selector.Targets {
				identity, err := target.Identity()
				s.Require().NoError(err)
				result = append(result, identity.KeyHash)
			}
		}

		return result
	}
	var expected []string
	for _, key := range []rte.Key{all, featureKey, meteredKey} {
		target, err := rte.NewCalculationRequestedEvent(key)
		s.Require().NoError(err)
		identity, err := target.Identity()
		s.Require().NoError(err)
		expected = append(expected, identity.KeyHash)
	}

	s.ElementsMatch(expected, hashes(selectors))
	resolvedMeter, err := s.MeterAdapter.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: ns, IDOrSlug: "api-requests-total"})
	s.Require().NoError(err)
	for _, fact := range []marshaler.Event{
		featurepkg.NewFeatureUpdateEvent(ctx, &feature.Feature),
		meter.NewMeterUpdateEvent(ctx, &resolvedMeter),
	} {
		selected, err := resolver.Resolve(ctx, fact)
		s.Require().NoError(err)
		s.ElementsMatch(expected, hashes(selected))
	}

	customerSelected, err := resolver.Resolve(ctx, customer.NewCustomerUpdateEvent(ctx, cust))
	s.Require().NoError(err)
	s.Len(hashes(customerSelected), 6)

	// then: unmatched subjects/meters and usage outside either source's period
	// do not schedule calculator work, and demand deactivation is respected.
	for _, change := range []func(*ingestevents.EventBatchedIngest){
		func(e *ingestevents.EventBatchedIngest) { e.SubjectKey = "untracked-subject" },
		func(e *ingestevents.EventBatchedIngest) { e.MeterSlugs = []string{"unrelated-meter"} },
		func(e *ingestevents.EventBatchedIngest) {
			e.RawEvents = []serializer.CloudEventsKafkaPayload{{Time: start.Add(-time.Hour).Unix()}}
		},
	} {
		changed := event
		change(&changed)
		selected, err := resolver.Resolve(ctx, changed)
		s.Require().NoError(err)
		s.Empty(selected)
	}

	for _, key := range []rte.Key{all, featureKey, meteredKey} {
		target, err := rte.NewCalculationRequestedEvent(key)
		s.Require().NoError(err)
		s.Require().NoError(repository.Deactivate(ctx, target.CalculationTarget))
	}

	selectors, err = resolver.Resolve(ctx, event)
	s.Require().NoError(err)
	s.Empty(selectors)
}
