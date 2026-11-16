package productcatalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// stubTaxCodeResolver serves BatchResolve from an in-memory map and records each call.
type stubTaxCodeResolver struct {
	taxCodes map[string]*taxcode.TaxCode
	err      error
	calls    [][]string
}

var _ productcatalog.TaxCodeResolver = (*stubTaxCodeResolver)(nil)

func (s *stubTaxCodeResolver) Resolve(context.Context, string, string) (*taxcode.TaxCode, error) {
	return nil, errors.New("not implemented")
}

func (s *stubTaxCodeResolver) BatchResolve(_ context.Context, _ string, ids ...string) (map[string]*taxcode.TaxCode, error) {
	s.calls = append(s.calls, ids)

	if s.err != nil {
		return nil, s.err
	}

	return lo.PickByKeys(s.taxCodes, ids), nil
}

func (s *stubTaxCodeResolver) WithNamespace(string) productcatalog.NamespacedTaxCodeResolver {
	return nil
}

func newFlatRateCard(key string, cfg *productcatalog.TaxCodeConfig) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{Key: key, Name: key, TaxConfig: cfg},
	}
}

// ValidateRateCardTaxCodes pins the single-call validation: missing IDs are rejected as
// rate_card_tax_code_not_found, soft-deleted ones as rate_card_tax_code_deleted unless the caller
// allows them, and resolver failures stay plain errors.
func TestValidateRateCardTaxCodes(t *testing.T) {
	const (
		namespace = "ns"
		liveID    = "01JTAXC0DE0000000000000000"
		deletedID = "01JTAXC0DE0000000000000001"
		missingID = "01JTAXC0DE0000000000000002"
	)

	taxCodes := map[string]*taxcode.TaxCode{
		liveID: {NamespacedID: models.NamespacedID{Namespace: namespace, ID: liveID}},
		deletedID: {
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deletedID},
			ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(clock.Now().Add(-1))},
		},
	}

	tests := []struct {
		name          string
		rateCards     productcatalog.RateCards
		allowDeleted  []string
		resolverErr   error
		wantCalls     [][]string
		wantNotFound  bool
		wantDeleted   bool
		wantErrNonVal bool
	}{
		{
			name:      "no tax configs skip the lookup",
			rateCards: productcatalog.RateCards{newFlatRateCard("a", nil)},
		},
		{
			name: "behavior only skips the lookup",
			rateCards: productcatalog.RateCards{
				newFlatRateCard("a", &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)}),
			},
		},
		{
			name:      "existing tax code",
			rateCards: productcatalog.RateCards{newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: liveID})},
			wantCalls: [][]string{{liveID}},
		},
		{
			name:         "missing tax code",
			rateCards:    productcatalog.RateCards{newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: missingID})},
			wantCalls:    [][]string{{missingID}},
			wantNotFound: true,
		},
		{
			name:        "deleted tax code",
			rateCards:   productcatalog.RateCards{newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: deletedID})},
			wantCalls:   [][]string{{deletedID}},
			wantDeleted: true,
		},
		{
			name:         "deleted tax code allowed passes",
			rateCards:    productcatalog.RateCards{newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: deletedID})},
			allowDeleted: []string{deletedID},
			wantCalls:    [][]string{{deletedID}},
		},
		{
			name:         "missing tax code listed as allowed is still not found",
			rateCards:    productcatalog.RateCards{newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: missingID})},
			allowDeleted: []string{missingID},
			wantCalls:    [][]string{{missingID}},
			wantNotFound: true,
		},
		{
			name:          "resolver failure is not a validation error",
			rateCards:     productcatalog.RateCards{newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: liveID})},
			resolverErr:   errors.New("db down"),
			wantCalls:     [][]string{{liveID}},
			wantErrNonVal: true,
		},
		{
			name: "shared tax code is looked up once",
			rateCards: productcatalog.RateCards{
				newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: liveID}),
				newFlatRateCard("b", &productcatalog.TaxCodeConfig{TaxCodeID: liveID}),
			},
			wantCalls: [][]string{{liveID}},
		},
		{
			name: "deleted and missing in one lookup",
			rateCards: productcatalog.RateCards{
				newFlatRateCard("a", &productcatalog.TaxCodeConfig{TaxCodeID: deletedID}),
				newFlatRateCard("b", &productcatalog.TaxCodeConfig{TaxCodeID: missingID}),
			},
			wantCalls:    [][]string{{deletedID, missingID}},
			wantNotFound: true,
			wantDeleted:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &stubTaxCodeResolver{taxCodes: taxCodes, err: tt.resolverErr}

			err := productcatalog.ValidateRateCardTaxCodes(t.Context(), resolver, namespace, tt.rateCards, tt.allowDeleted...)

			assert.Equal(t, tt.wantCalls, resolver.calls)

			switch {
			case tt.wantNotFound:
				require.Error(t, err)
				assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
				assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)
			case tt.wantDeleted:
				require.Error(t, err)
				assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
				assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeDeleted)
			case tt.wantErrNonVal:
				require.Error(t, err)
				assert.False(t, models.IsGenericValidationError(err), "resolver failures must not become validation errors")
				assert.ErrorIs(t, err, tt.resolverErr)
			default:
				require.NoError(t, err)
			}
		})
	}
}

// ValidatePlanWithTaxCodes pins the plan publish gate: deleted and missing references from any
// phase are rejected in one batch lookup, each issue prefixed with its phase, and lookup
// failures are returned as is.
func TestValidatePlanWithTaxCodes(t *testing.T) {
	const (
		namespace     = "ns"
		liveID        = "01JTAXC0DE0000000000000000"
		deletedID     = "01JTAXC0DE0000000000000001"
		missingID     = "01JTAXC0DE0000000000000002"
		untouchedLive = "01JTAXC0DE0000000000000003"
	)

	taxCodes := map[string]*taxcode.TaxCode{
		liveID:        {NamespacedID: models.NamespacedID{Namespace: namespace, ID: liveID}},
		untouchedLive: {NamespacedID: models.NamespacedID{Namespace: namespace, ID: untouchedLive}},
		deletedID: {
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deletedID},
			ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(clock.Now().Add(-1))},
		},
	}

	// given: a plan whose phases reference a live, a deleted and a missing Tax Code
	resolver := &stubTaxCodeResolver{taxCodes: taxCodes}

	plan := productcatalog.Plan{
		PlanMeta: productcatalog.PlanMeta{Key: "plan", Name: "Plan"},
		Phases: []productcatalog.Phase{
			{
				PhaseMeta: productcatalog.PhaseMeta{Key: "phase-1", Name: "Phase 1"},
				RateCards: productcatalog.RateCards{
					newFlatRateCard("untouched", &productcatalog.TaxCodeConfig{TaxCodeID: untouchedLive}),
				},
			},
			{
				PhaseMeta: productcatalog.PhaseMeta{Key: "phase-2", Name: "Phase 2"},
				RateCards: productcatalog.RateCards{
					newFlatRateCard("deleted", &productcatalog.TaxCodeConfig{TaxCodeID: deletedID}),
					newFlatRateCard("missing", &productcatalog.TaxCodeConfig{TaxCodeID: missingID}),
				},
			},
		},
	}

	// when: the plan is validated across phases holding deleted and missing codes
	err := productcatalog.ValidatePlanWithTaxCodes(t.Context(), resolver, namespace)(plan)

	// then: both issues surface, prefixed with their phase, from one batch lookup
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeDeleted)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)

	require.Len(t, resolver.calls, 1, "one BatchResolve must cover all phases")
	assert.ElementsMatch(t, []string{untouchedLive, deletedID, missingID}, resolver.calls[0])

	issues, convErr := models.AsValidationIssues(err)
	require.NoError(t, convErr)
	require.Len(t, issues, 2)

	byCode := make(map[models.ErrorCode]models.ValidationIssue, len(issues))
	for _, issue := range issues {
		byCode[issue.Code()] = issue
	}

	deletedIssue, ok := byCode[productcatalog.ErrCodeRateCardTaxCodeDeleted]
	require.True(t, ok, "deleted issue must be reported, got issues: %v", issues)
	assert.Equal(t, "tax code deleted", deletedIssue.Message())
	assert.Contains(t, deletedIssue.Field().String(), `phases[key=phase-2]`)
	assert.Contains(t, deletedIssue.Field().String(), `ratecards[key=deleted]`)
	assert.Contains(t, deletedIssue.Field().String(), "taxConfig")

	missingIssue, ok := byCode[productcatalog.ErrCodeRateCardTaxCodeNotFound]
	require.True(t, ok, "missing issue must be reported, got issues: %v", issues)
	assert.Contains(t, missingIssue.Field().String(), `phases[key=phase-2]`)
	assert.Contains(t, missingIssue.Field().String(), `ratecards[key=missing]`)
}

// A failed lookup is returned as is, not wrapped as a validation error, so it stays a 500.
func TestValidatePlanWithTaxCodesLookupError(t *testing.T) {
	// given: a resolver that fails
	resolverErr := errors.New("db is down")
	resolver := &stubTaxCodeResolver{err: resolverErr}

	plan := productcatalog.Plan{
		PlanMeta: productcatalog.PlanMeta{Key: "plan", Name: "Plan"},
		Phases: []productcatalog.Phase{
			{
				PhaseMeta: productcatalog.PhaseMeta{Key: "phase-1", Name: "Phase 1"},
				RateCards: productcatalog.RateCards{
					newFlatRateCard("rc", &productcatalog.TaxCodeConfig{TaxCodeID: "01JTAXC0DE0000000000000000"}),
				},
			},
		},
	}

	// when: the plan is validated
	err := productcatalog.ValidatePlanWithTaxCodes(t.Context(), resolver, "ns")(plan)

	// then: the lookup failure is passed through, not turned into a validation error
	require.Error(t, err)
	assert.False(t, models.IsGenericValidationError(err), "resolver failures must not become validation errors")
	assert.ErrorIs(t, err, resolverErr)
}

// ValidateAddonWithTaxCodes pins the add-on publish gate: deleted and missing references are
// rejected in one batch lookup, each issue prefixed with its rate card, and lookup failures are
// returned as is.
func TestValidateAddonWithTaxCodes(t *testing.T) {
	const (
		namespace = "ns"
		liveID    = "01JTAXC0DE0000000000000000"
		deletedID = "01JTAXC0DE0000000000000001"
		missingID = "01JTAXC0DE0000000000000002"
	)

	taxCodes := map[string]*taxcode.TaxCode{
		liveID: {NamespacedID: models.NamespacedID{Namespace: namespace, ID: liveID}},
		deletedID: {
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deletedID},
			ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(clock.Now().Add(-1))},
		},
	}

	// given: an add-on whose rate cards reference a live, a deleted and a missing Tax Code
	resolver := &stubTaxCodeResolver{taxCodes: taxCodes}

	addon := productcatalog.Addon{
		AddonMeta: productcatalog.AddonMeta{Key: "addon", Name: "Addon"},
		RateCards: productcatalog.RateCards{
			newFlatRateCard("live", &productcatalog.TaxCodeConfig{TaxCodeID: liveID}),
			newFlatRateCard("deleted", &productcatalog.TaxCodeConfig{TaxCodeID: deletedID}),
			newFlatRateCard("missing", &productcatalog.TaxCodeConfig{TaxCodeID: missingID}),
		},
	}

	// when: the add-on is validated
	err := productcatalog.ValidateAddonWithTaxCodes(t.Context(), resolver, namespace)(addon)

	// then: both issues surface, prefixed with their rate card, from one batch lookup
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeDeleted)
	assert.ErrorIs(t, err, productcatalog.ErrRateCardTaxCodeNotFound)

	require.Len(t, resolver.calls, 1, "one BatchResolve must cover all rate cards")
	assert.ElementsMatch(t, []string{liveID, deletedID, missingID}, resolver.calls[0])

	issues, convErr := models.AsValidationIssues(err)
	require.NoError(t, convErr)
	require.Len(t, issues, 2)

	byCode := make(map[models.ErrorCode]models.ValidationIssue, len(issues))
	for _, issue := range issues {
		byCode[issue.Code()] = issue
	}

	deletedIssue, ok := byCode[productcatalog.ErrCodeRateCardTaxCodeDeleted]
	require.True(t, ok, "deleted issue must be reported, got issues: %v", issues)
	assert.Equal(t, "tax code deleted", deletedIssue.Message())
	assert.Contains(t, deletedIssue.Field().String(), `ratecards[key=deleted]`)
	assert.Contains(t, deletedIssue.Field().String(), "taxConfig")

	missingIssue, ok := byCode[productcatalog.ErrCodeRateCardTaxCodeNotFound]
	require.True(t, ok, "missing issue must be reported, got issues: %v", issues)
	assert.Contains(t, missingIssue.Field().String(), `ratecards[key=missing]`)
}

// A failed lookup is returned as is, not wrapped as a validation error, so it stays a 500.
func TestValidateAddonWithTaxCodesLookupError(t *testing.T) {
	// given: a resolver that fails
	resolverErr := errors.New("db is down")
	resolver := &stubTaxCodeResolver{err: resolverErr}

	addon := productcatalog.Addon{
		AddonMeta: productcatalog.AddonMeta{Key: "addon", Name: "Addon"},
		RateCards: productcatalog.RateCards{
			newFlatRateCard("rc", &productcatalog.TaxCodeConfig{TaxCodeID: "01JTAXC0DE0000000000000000"}),
		},
	}

	// when: the add-on is validated
	err := productcatalog.ValidateAddonWithTaxCodes(t.Context(), resolver, "ns")(addon)

	// then: the lookup failure is passed through, not turned into a validation error
	require.Error(t, err)
	assert.False(t, models.IsGenericValidationError(err), "resolver failures must not become validation errors")
	assert.ErrorIs(t, err, resolverErr)
}
