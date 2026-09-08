package chargeadapter_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestLoadLineagesFiltersBackfillCandidates(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	ctx := t.Context()
	adapter, err := lineageadapter.New(lineageadapter.Config{Client: env.DB})
	require.NoError(t, err)

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
		namespace, customerID, currency := env.Namespace, env.CustomerID.ID, env.currency.Reference()
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
		_, err := env.DB.Charge.Create().SetID(chargeID).SetNamespace(namespace).SetType(meta.ChargeTypeUsageBased).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, adapter.CreateLineages(ctx, lineage.CreateLineagesInput{
			Namespace: namespace, CustomerID: customerID, ChargeID: chargeID, Currency: currency,
			Specs: []creditrealization.InitialLineageSpec{{
				LineageID: rootID, RootRealizationID: ulid.Make().String(), OriginKind: origin,
				InitialState: creditrealization.InitialLineageSegmentState(origin),
				Amount:       alpacadecimal.NewFromInt(20), AdvanceFeatures: scenario.features,
			}},
		}))
		ids[scenario.name] = rootID
		segments, err := adapter.ListActiveSegments(ctx, lineage.ListActiveSegmentsInput{LineageIDs: []string{rootID}})
		require.NoError(t, err)
		require.Len(t, segments, 1)
		if scenario.backed > 0 {
			require.NoError(t, env.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
				Namespace: namespace, CustomerID: customerID, Currency: env.currency,
				Amount: alpacadecimal.NewFromInt(scenario.backed), BackingTransactionGroupID: ulid.Make().String(),
				Allocations: []lineage.AdvanceBackfillAllocation{{SegmentID: segments[0].ID, Amount: alpacadecimal.NewFromInt(scenario.backed)}},
			}))
		}
		if scenario.consumed {
			require.NoError(t, env.lineage.CloseSegment(ctx, segments[0].ID, env.Now()))
		}
	}

	for _, scenario := range []struct {
		name     string
		input    lineage.LoadLineagesByCustomerInput
		expected []string
	}{
		{name: "default preserves history", expected: []string{"api", "partial", "storage", "featureless", "fully backed", "consumed", "real credit"}},
		{name: "active excludes consumed roots", input: lineage.LoadLineagesByCustomerInput{HasActiveSegments: true}, expected: []string{"api", "partial", "storage", "featureless", "fully backed", "real credit"}},
		{name: "origin selects ordinary credit", input: lineage.LoadLineagesByCustomerInput{OriginKind: lo.ToPtr(creditrealization.LineageOriginKindRealCredit)}, expected: []string{"real credit"}},
		{name: "uncovered excludes fully backed and consumed roots", input: lineage.LoadLineagesByCustomerInput{OriginKind: lo.ToPtr(creditrealization.LineageOriginKindAdvance), SegmentState: lo.ToPtr(creditrealization.LineageSegmentStateAdvanceUncovered)}, expected: []string{"api", "partial", "storage", "featureless"}},
		{name: "restricted purchase includes any overlapping feature", input: lineage.LoadLineagesByCustomerInput{OriginKind: lo.ToPtr(creditrealization.LineageOriginKindAdvance), HasActiveSegments: true, SegmentState: lo.ToPtr(creditrealization.LineageSegmentStateAdvanceUncovered), FeatureFilters: []string{"api", "unrelated"}}, expected: []string{"api", "partial"}},
		{name: "no matching feature", input: lineage.LoadLineagesByCustomerInput{FeatureFilters: []string{"unrelated"}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// When the caller selects candidate roots in the database.
			input := scenario.input
			input.Namespace, input.CustomerID, input.Currency = env.Namespace, env.CustomerID.ID, env.currency.Reference()
			roots, err := env.lineage.LoadLineagesByCustomer(ctx, input)
			require.NoError(t, err)

			// Then excluded history contributes neither roots nor eager-loaded segments.
			actual := lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID })
			expected := lo.Map(scenario.expected, func(name string, _ int) string { return ids[name] })
			require.ElementsMatch(t, expected, actual)
			if input.SegmentState != nil {
				for _, root := range roots {
					require.Len(t, root.Segments, 1)
					require.Equal(t, *input.SegmentState, root.Segments[0].State)
					if root.ID == ids["partial"] {
						require.Equal(t, float64(15), root.Segments[0].Amount.InexactFloat64())
					}
				}
			}
		})
	}
}
