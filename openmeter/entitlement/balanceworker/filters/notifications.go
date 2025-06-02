package filters

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/lrux"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*NotificationsFilterConfig)(nil)

type NotificationsFilterConfig struct {
	NotificationService notification.Service
	CacheTTL            time.Duration
	CacheSize           int
}

func (c NotificationsFilterConfig) Validate() error {
	if c.NotificationService == nil {
		return fmt.Errorf("notificationService is required")
	}

	if c.CacheTTL <= 0 {
		return fmt.Errorf("cache ttl must be positive")
	}

	if c.CacheSize <= 0 {
		return fmt.Errorf("cache size must be positive")
	}

	return nil
}

var _ Filter = (*NotificationsFilter)(nil)

type NotificationsFilter struct {
	notificationService notification.Service

	ruleCache *lrux.CacheWithItemTTL[string, []notification.Rule]
}

func NewNotificationsFilter(cfg NotificationsFilterConfig) (NamedFilter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	filter := &NotificationsFilter{
		notificationService: cfg.NotificationService,
	}

	ruleCache, err := lrux.NewCacheWithItemTTL(cfg.CacheSize, filter.fetchRulesForNamespace, lrux.WithTTL(cfg.CacheTTL))
	if err != nil {
		return nil, err
	}

	filter.ruleCache = ruleCache

	return filter, nil
}

func (f *NotificationsFilter) Name() string {
	return "notification_rules"
}

func (f *NotificationsFilter) IsNamespaceInScope(ctx context.Context, namespace string) (bool, error) {
	rules, err := f.ruleCache.Get(ctx, namespace)
	if err != nil {
		return false, err
	}

	return len(rules) > 0, nil
}

func (f *NotificationsFilter) IsEntitlementInScope(ctx context.Context, ent entitlement.Entitlement) (bool, error) {
	rules, err := f.ruleCache.Get(ctx, ent.Namespace)
	if err != nil {
		return false, err
	}

	for _, rule := range rules {
		if rule.Config.BalanceThreshold == nil {
			continue
		}

		if len(rule.Config.BalanceThreshold.Features) == 0 {
			// Active for all features => entitlement is in scope
			return true, nil
		}

		if lo.Contains(rule.Config.BalanceThreshold.Features, ent.FeatureKey) {
			return true, nil
		}

		if lo.Contains(rule.Config.BalanceThreshold.Features, ent.FeatureID) {
			return true, nil
		}
	}

	return false, nil
}

func (f *NotificationsFilter) fetchRulesForNamespace(ctx context.Context, namespace string) ([]notification.Rule, error) {
	rulesPage, err := f.notificationService.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{namespace},
		Types: []notification.EventType{
			notification.EventTypeBalanceThreshold,
		},
	})
	if err != nil {
		return nil, err
	}

	return rulesPage.Items, nil
}
