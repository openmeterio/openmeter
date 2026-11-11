package rte_test

import (
	"encoding/json"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *RTEIntegrationSuite) TestMeteredEntitlementCalculateStoreAndRead() {
	ctx := s.T().Context()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// Given a real metered entitlement with 100 granted units and usage in its first period.
	ns := s.GetUniqueNamespace("rte-metered")
	cust := s.CreateTestCustomer(ns, "rte-metered-customer")
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()

	ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
		CustomerID: cust.GetID(),
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID:       &feature.Feature.ID,
			EntitlementType: entitlement.EntitlementTypeMetered,
			IssueAfterReset: lo.ToPtr(float64(100)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor: start, Interval: timeutil.RecurrencePeriodMonth,
			})),
		},
	})
	s.Require().NoError(err)
	s.MockStreamingConnector.AddSimpleEvent("api-requests-total", 25, start.Add(time.Minute))

	key := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}
	pipeline := s.newEventPipeline()
	engine := pipeline.service
	_, err = engine.MeteredEntitlements().Get(ctx, key)
	s.True(models.IsGenericNotFoundError(err))

	// When explicitly calculated, RTE preserves the existing calculator's complete result.
	now := start.Add(2*time.Minute + 15*time.Second + 123456789*time.Nanosecond)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	expected, err := s.Entitlements.MeteredEntitlement.GetEntitlementBalance(ctx, key.EntitlementID, now)
	s.Require().NoError(err)
	updated := pipeline.calculate(s.T(), key)
	var calculated rte.State[rtemetered.Balance]
	s.Require().NoError(json.Unmarshal(updated.State, &calculated))
	storedAfterEvent, err := engine.MeteredEntitlements().Get(ctx, key)
	s.Require().NoError(err)
	s.True(calculated.CalculatedAt.Equal(storedAfterEvent.CalculatedAt))
	s.Equal(calculated.Value, storedAfterEvent.Value)
	s.Equal(*expected, calculated.Value)
	s.Equal(float64(75), calculated.Value.Balance)
	s.Equal(float64(25), calculated.Value.UsageInPeriod)
	s.Equal(float64(100), calculated.Value.TotalAvailableGrantAmount)
	s.Len(calculated.Value.GrantBalances, 1)
	s.True(calculated.Value.StartOfPeriod.Equal(start))
	s.True(calculated.CalculatedAt.Equal(now.Truncate(time.Microsecond)))

	// Then another service reads the stored result even after fresh usage arrives.
	s.MockStreamingConnector.AddSimpleEvent("api-requests-total", 90, start.Add(3*time.Minute))
	reader := s.newRTE()
	stored, err := reader.MeteredEntitlements().Get(ctx, key)
	s.Require().NoError(err)
	s.True(calculated.CalculatedAt.Equal(stored.CalculatedAt))
	s.Equal(calculated.Value, stored.Value)

	clock.FreezeTime(start.Add(4 * time.Minute))
	defer clock.UnFreeze()
	refreshed, err := reader.MeteredEntitlements().Calculate(ctx, key)
	s.Require().NoError(err)
	s.Equal(float64(0), refreshed.Value.Balance)
	s.Equal(float64(115), refreshed.Value.UsageInPeriod)
	s.Equal(float64(15), refreshed.Value.Overage)
	s.True(refreshed.CalculatedAt.After(stored.CalculatedAt))
}
