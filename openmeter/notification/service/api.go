package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s Service) ListRuleViews(ctx context.Context, params notification.ListRulesInput) (pagination.Result[notification.RuleView], error) {
	result, err := s.ListRules(ctx, params)
	if err != nil {
		return pagination.Result[notification.RuleView]{}, err
	}

	features, err := s.resolveRuleFeatures(ctx, params.Namespace, lo.FlatMap(result.Items, func(r notification.Rule, _ int) []string {
		return r.Config.Features()
	}))
	if err != nil {
		return pagination.Result[notification.RuleView]{}, err
	}

	views := s.mergeRulesFeatures(result.Items, features)

	return pagination.Result[notification.RuleView]{Page: result.Page, TotalCount: result.TotalCount, Items: views}, nil
}

func (s Service) GetRuleView(ctx context.Context, params notification.GetRuleInput) (notification.RuleView, error) {
	rule, err := s.GetRule(ctx, params)
	if err != nil {
		return notification.RuleView{}, err
	}

	if rule == nil {
		return notification.RuleView{}, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}}
	}

	features, err := s.resolveRuleFeatures(ctx, params.Namespace, rule.Config.Features())
	if err != nil {
		return notification.RuleView{}, err
	}

	return s.mergeRuleFeatures(*rule, features), nil
}

func (s Service) CreateRuleView(ctx context.Context, params notification.CreateRuleInput) (notification.RuleView, error) {
	features, err := s.resolveRuleFeatures(ctx, params.Namespace, params.Config.Features())
	if err != nil {
		return notification.RuleView{}, err
	}

	rule, err := s.CreateRule(ctx, params)
	if err != nil {
		return notification.RuleView{}, err
	}

	if rule == nil {
		return notification.RuleView{}, errors.New("unable to create rule")
	}

	return s.mergeRuleFeatures(*rule, features), nil
}

func (s Service) UpdateRuleView(ctx context.Context, params notification.UpdateRuleInput) (notification.RuleView, error) {
	features, err := s.resolveRuleFeatures(ctx, params.Namespace, params.Config.Features())
	if err != nil {
		return notification.RuleView{}, err
	}

	rule, err := s.UpdateRule(ctx, params)
	if err != nil {
		return notification.RuleView{}, err
	}

	if rule == nil {
		return notification.RuleView{}, errors.New("unable to update rule")
	}

	return s.mergeRuleFeatures(*rule, features), nil
}

// Archived features are included so views stay faithful to the stored rule; when a
// key was reused after archiving, the live feature wins. Writes resolve before
// mutating so a failed lookup never surfaces as an error for an already committed
// rule; missing features are rejected by the rule validation before any write.
func (s Service) resolveRuleFeatures(ctx context.Context, namespace string, idsOrKeys []string) ([]feature.Feature, error) {
	idsOrKeys = lo.Uniq(idsOrKeys)
	if len(idsOrKeys) == 0 {
		return nil, nil
	}

	features, err := s.feature.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:       namespace,
		IDsOrKeys:       idsOrKeys,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notification rule features: %w", err)
	}

	featuresByIDOrKey := make(map[string]feature.Feature, 2*len(features.Items))
	for _, f := range features.Items {
		featuresByIDOrKey[f.ID] = f

		if existing, ok := featuresByIDOrKey[f.Key]; !ok || existing.ArchivedAt != nil {
			featuresByIDOrKey[f.Key] = f
		}
	}

	resolved := lo.FilterMap(idsOrKeys, func(idOrKey string, _ int) (feature.Feature, bool) {
		f, ok := featuresByIDOrKey[idOrKey]
		return f, ok
	})

	return lo.UniqBy(resolved, func(f feature.Feature) string { return f.ID }), nil
}

func (s Service) mergeRulesFeatures(rules []notification.Rule, features []feature.Feature) []notification.RuleView {
	return lo.Map(rules, func(r notification.Rule, _ int) notification.RuleView {
		return s.mergeRuleFeatures(r, features)
	})
}

func (s Service) mergeRuleFeatures(rule notification.Rule, features []feature.Feature) notification.RuleView {
	ruleViewFeatures := lo.Filter(features, func(feature feature.Feature, _ int) bool {
		return lo.ContainsBy(rule.Config.Features(), func(idOrKey string) bool {
			return idOrKey == feature.ID || idOrKey == feature.Key
		})
	})

	return notification.RuleView{
		Rule:     rule,
		Features: lo.UniqBy(ruleViewFeatures, func(f feature.Feature) string { return f.ID }),
	}
}
