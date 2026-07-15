package meteredentitlement_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestGetEntitlementBalanceWithUnitConfig is the OM-400 acceptance test: with a
// divide-by-1e9 unit_config and a 100-unit grant (a 100 GB quota, authored in
// converted units), a customer who has consumed 99.3 GB of raw bytes is still
// granted access with 0.7 GB remaining, while the meter recorded the raw 99.3e9.
// Rounding belongs to invoicing; balance checks use the precise converted value.
func TestGetEntitlementBalanceWithUnitConfig(t *testing.T) {
	namespace := "ns1"

	// UnitConfig enabled: the grant-owner adapter converts metered usage before burndown.
	connector, deps := setupConnector(t, withUnitConfigEnabled())
	defer deps.Teardown()

	// divide-by-1e9 converts raw bytes into GB for the entitlement quota.
	unitCfg := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1_000_000_000),
	}

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature-1",
		MeterID:             &deps.meterID,
		MeterGroupByFilters: map[string]filter.FilterString{},
	})
	require.NoError(t, err)

	ctx := t.Context()
	startTime := getAnchor(t)

	randName := testutils.NameGenerator.Generate()
	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

	// given:
	// - a metered entitlement whose limit is authored in converted units (GB),
	//   carrying a divide-by-1e9 unit_config snapshot
	// - a 100-unit grant = a 100 GB quota
	inp := entitlement.CreateEntitlementRepoInputs{
		Namespace:        namespace,
		FeatureID:        feat.ID,
		FeatureKey:       feat.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		MeasureUsageFrom: &startTime,
		EntitlementType:  entitlement.EntitlementTypeMetered,
		IssueAfterReset:  convert.ToPointer(0.0),
		IsSoftLimit:      convert.ToPointer(false),
		UnitConfig:       unitCfg,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Anchor:   getAnchor(t),
			Interval: timeutil.RecurrencePeriodYear,
		})),
	}
	currentUsagePeriod, err := inp.UsagePeriod.GetValue().GetPeriodAt(time.Now())
	require.NoError(t, err)
	inp.CurrentUsagePeriod = &currentUsagePeriod

	ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
	require.NoError(t, err)

	_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
		OwnerID:     ent.ID,
		Namespace:   namespace,
		Amount:      100,
		Priority:    1,
		EffectiveAt: startTime,
		ExpiresAt:   lo.ToPtr(startTime.AddDate(1, 0, 0)),
	})
	require.NoError(t, err)

	// when:
	// - the customer has consumed 99.3 GB of raw bytes (99.3e9)
	deps.streamingConnector.AddSimpleEvent(meterSlug, 99_300_000_000, startTime.Add(time.Minute))

	queryTime := startTime.Add(time.Hour)
	entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, queryTime)
	require.NoError(t, err)

	// then:
	// - usage and balance are expressed in converted units: 99.3 GB used, 0.7 GB left
	// - access is still granted (balance > 0, no overage)
	assert.InDelta(t, 99.3, entBalance.UsageInPeriod, 1e-6)
	assert.InDelta(t, 0.7, entBalance.Balance, 1e-6)
	assert.Equal(t, 0.0, entBalance.Overage)
}
