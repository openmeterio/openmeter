package service

import (
	"context"
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

	views, err := s.resolveRuleViews(ctx, params.Namespace, result.Items)
	if err != nil {
		return pagination.Result[notification.RuleView]{}, err
	}

	return pagination.Result[notification.RuleView]{Page: result.Page, TotalCount: result.TotalCount, Items: views}, nil
}

func (s Service) GetRuleView(ctx context.Context, params notification.GetRuleInput) (notification.RuleView, error) {
	rule, err := s.GetRule(ctx, params)
	if err != nil {
		return notification.RuleView{}, err
	}

	return s.resolveRuleView(ctx, rule)
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
		return notification.RuleView{}, fmt.Errorf("nil rule returned")
	}

	return notification.RuleView{Rule: *rule, Features: features}, nil
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
		return notification.RuleView{}, fmt.Errorf("nil rule returned")
	}

	return notification.RuleView{Rule: *rule, Features: features}, nil
}

// Writes resolve their features up front so a missing feature or a failed lookup
// rejects the request before anything is committed to the database or Svix.
func (s Service) resolveRuleFeatures(ctx context.Context, namespace string, idsOrKeys []string) ([]feature.Feature, error) {
	idsOrKeys = lo.Uniq(idsOrKeys)
	if len(idsOrKeys) == 0 {
		return nil, nil
	}

	features, err := s.ListFeature(ctx, namespace, idsOrKeys...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notification rule features: %w", err)
	}

	featuresByIDOrKey := make(map[string]feature.Feature, 2*len(features))
	for _, f := range features {
		featuresByIDOrKey[f.ID] = f
		featuresByIDOrKey[f.Key] = f
	}

	var missing []string

	resolved := make([]feature.Feature, 0, len(idsOrKeys))

	for _, idOrKey := range idsOrKeys {
		f, ok := featuresByIDOrKey[idOrKey]
		if !ok {
			missing = append(missing, idOrKey)
			continue
		}

		resolved = append(resolved, f)
	}

	if len(missing) > 0 {
		return nil, models.NewGenericValidationError(fmt.Errorf("non-existing features: %v", missing))
	}

	return lo.UniqBy(resolved, func(f feature.Feature) string { return f.ID }), nil
}

func (s Service) resolveRuleView(ctx context.Context, rule *notification.Rule) (notification.RuleView, error) {
	if rule == nil {
		return notification.RuleView{}, fmt.Errorf("nil rule returned")
	}

	views, err := s.resolveRuleViews(ctx, rule.Namespace, []notification.Rule{*rule})
	if err != nil {
		return notification.RuleView{}, err
	}

	return views[0], nil
}

// Archived features are included to keep the view faithful to the stored rule; when a
// key was reused after archiving, the live feature wins.
func (s Service) resolveRuleViews(ctx context.Context, namespace string, rules []notification.Rule) ([]notification.RuleView, error) {
	idsOrKeys := lo.Uniq(lo.FlatMap(rules, func(r notification.Rule, _ int) []string {
		return r.Config.Features()
	}))

	featuresByIDOrKey := make(map[string]feature.Feature, 2*len(idsOrKeys))

	if len(idsOrKeys) > 0 {
		features, err := s.feature.ListFeatures(ctx, feature.ListFeaturesParams{
			Namespace:       namespace,
			IDsOrKeys:       idsOrKeys,
			IncludeArchived: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve notification rule features: %w", err)
		}

		for _, f := range features.Items {
			featuresByIDOrKey[f.ID] = f

			if existing, ok := featuresByIDOrKey[f.Key]; !ok || existing.ArchivedAt != nil {
				featuresByIDOrKey[f.Key] = f
			}
		}
	}

	return lo.Map(rules, func(r notification.Rule, _ int) notification.RuleView {
		resolved := lo.FilterMap(r.Config.Features(), func(idOrKey string, _ int) (feature.Feature, bool) {
			f, ok := featuresByIDOrKey[idOrKey]
			return f, ok
		})

		return notification.RuleView{
			Rule:     r,
			Features: lo.UniqBy(resolved, func(f feature.Feature) string { return f.ID }),
		}
	}), nil
}
