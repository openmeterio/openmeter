package adapter_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	legacylineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage/adapter"
	legacylineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestLoadLineagesFiltersBackfillCandidates(t *testing.T) {
	ctx := t.Context()
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() {
		require.NoError(t, testDB.EntDriver.Close())
		require.NoError(t, testDB.PGDriver.Close())
	})
	db := testDB.EntDriver.Client()
	adapter, err := legacylineageadapter.New(legacylineageadapter.Config{Client: db})
	require.NoError(t, err)
	service, err := legacylineageservice.New(legacylineageservice.Config{Adapter: adapter})
	require.NoError(t, err)
	testNamespace, testCustomerID := ulid.Make().String(), ulid.Make().String()
	testCurrency := currenciestestutils.NewFiatCurrency(t, "USD")

	// Given outstanding advances, a partial remainder, fully backed and consumed
	// history, ordinary credit, and unrelated customers/currencies/namespaces.
	ids := map[string]string{}
	for _, scenario := range []struct {
		name       string
		features   []string
		origin     creditrealization.LineageOriginKind
		backed     int64
		consumed   bool
		otherScope string
	}{
		{name: "api", features: []string{"api"}},
		{name: "partial", features: []string{"api", "storage"}, backed: 5},
		{name: "storage", features: []string{"storage"}},
		{name: "featureless"},
		{name: "fully backed", features: []string{"api"}, backed: 20},
		{name: "consumed", features: []string{"api"}, consumed: true},
		{name: "real credit", origin: creditrealization.LineageOriginKindRealCredit},
		{name: "other customer", features: []string{"api"}, otherScope: "customer"},
		{name: "other currency", features: []string{"api"}, otherScope: "currency"},
		{name: "other namespace", features: []string{"api"}, otherScope: "namespace"},
	} {
		namespace, customerID, currency := testNamespace, testCustomerID, testCurrency.Reference()
		switch scenario.otherScope {
		case "customer":
			customerID = ulid.Make().String()
		case "currency":
			currency = currencies.NewCurrencyReference(currencyx.Code("EUR"))
		case "namespace":
			namespace = ulid.Make().String()
		}
		origin := scenario.origin
		if origin == "" {
			origin = creditrealization.LineageOriginKindAdvance
		}
		chargeID, rootID := ulid.Make().String(), ulid.Make().String()
		_, err := db.Charge.Create().SetID(chargeID).SetNamespace(namespace).SetType(meta.ChargeTypeUsageBased).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, adapter.CreateLineages(ctx, legacylineage.CreateLineagesInput{
			Namespace: namespace, CustomerID: customerID, ChargeID: chargeID, Currency: currency,
			Specs: []creditrealization.InitialLineageSpec{{
				LineageID: rootID, RootRealizationID: ulid.Make().String(), OriginKind: origin,
				InitialState: creditrealization.InitialLineageSegmentState(origin),
				Amount:       alpacadecimal.NewFromInt(20), AdvanceFeatures: scenario.features,
			}},
		}))
		ids[scenario.name] = rootID
		segments, err := adapter.ListActiveSegments(ctx, legacylineage.ListActiveSegmentsInput{LineageIDs: []string{rootID}})
		require.NoError(t, err)
		require.Len(t, segments, 1)
		if scenario.backed > 0 {
			require.NoError(t, service.BackfillAdvanceLineageSegments(ctx, legacylineage.BackfillAdvanceLineageSegmentsInput{
				Namespace: namespace, CustomerID: customerID, Currency: testCurrency,
				Amount: alpacadecimal.NewFromInt(scenario.backed), BackingTransactionGroupID: ulid.Make().String(),
				Allocations: []legacylineage.AdvanceBackfillAllocation{{SegmentID: segments[0].ID, Amount: alpacadecimal.NewFromInt(scenario.backed)}},
			}))
		}
		if scenario.consumed {
			require.NoError(t, service.CloseSegment(ctx, segments[0].ID, clock.Now()))
		}
	}

	for _, scenario := range []struct {
		name     string
		input    legacylineage.LoadLineagesByCustomerInput
		expected []string
	}{
		{name: "default preserves history", expected: []string{"api", "partial", "storage", "featureless", "fully backed", "consumed", "real credit"}},
		{name: "active excludes consumed roots", input: legacylineage.LoadLineagesByCustomerInput{HasActiveSegments: true}, expected: []string{"api", "partial", "storage", "featureless", "fully backed", "real credit"}},
		{name: "origin selects ordinary credit", input: legacylineage.LoadLineagesByCustomerInput{OriginKind: lo.ToPtr(creditrealization.LineageOriginKindRealCredit)}, expected: []string{"real credit"}},
		{name: "uncovered excludes fully backed and consumed roots", input: legacylineage.LoadLineagesByCustomerInput{OriginKind: lo.ToPtr(creditrealization.LineageOriginKindAdvance), SegmentState: lo.ToPtr(creditrealization.LineageSegmentStateAdvanceUncovered)}, expected: []string{"api", "partial", "storage", "featureless"}},
		{name: "backfilled selects only backed segments", input: legacylineage.LoadLineagesByCustomerInput{SegmentState: lo.ToPtr(creditrealization.LineageSegmentStateAdvanceBackfilled)}, expected: []string{"partial", "fully backed"}},
		{name: "empty feature filter is unrestricted", input: legacylineage.LoadLineagesByCustomerInput{FeatureFilters: []string{}}, expected: []string{"api", "partial", "storage", "featureless", "fully backed", "consumed", "real credit"}},
		{name: "restricted purchase includes any overlapping feature", input: legacylineage.LoadLineagesByCustomerInput{OriginKind: lo.ToPtr(creditrealization.LineageOriginKindAdvance), HasActiveSegments: true, SegmentState: lo.ToPtr(creditrealization.LineageSegmentStateAdvanceUncovered), FeatureFilters: []string{"api", "unrelated"}}, expected: []string{"api", "partial"}},
		{name: "no matching feature", input: legacylineage.LoadLineagesByCustomerInput{FeatureFilters: []string{"unrelated"}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// When the caller selects candidate roots in the database.
			input := scenario.input
			input.Namespace, input.CustomerID, input.Currency = testNamespace, testCustomerID, testCurrency.Reference()
			roots, err := service.LoadLineagesByCustomer(ctx, input)
			require.NoError(t, err)

			// Then excluded history contributes neither roots nor eager-loaded segments.
			actual := lo.Map(roots, func(root legacylineage.Lineage, _ int) string { return root.ID })
			expected := lo.Map(scenario.expected, func(name string, _ int) string { return ids[name] })
			require.ElementsMatch(t, expected, actual)
			if input.SegmentState != nil {
				for _, root := range roots {
					require.Len(t, root.Segments, 1)
					require.Equal(t, *input.SegmentState, root.Segments[0].State)
					if root.ID == ids["partial"] {
						expectedAmount := float64(15)
						if *input.SegmentState == creditrealization.LineageSegmentStateAdvanceBackfilled {
							expectedAmount = 5
						}
						require.Equal(t, expectedAmount, root.Segments[0].Amount.InexactFloat64())
					}
				}
			}
		})
	}
}
