package notification

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// RuleAPIService is the API-facing facade for rules; handlers only translate types.
type RuleAPIService interface {
	ListRuleViews(ctx context.Context, params ListRulesInput) (pagination.Result[RuleView], error)
	GetRuleView(ctx context.Context, params GetRuleInput) (RuleView, error)
	CreateRuleView(ctx context.Context, params CreateRuleInput) (RuleView, error)
	UpdateRuleView(ctx context.Context, params UpdateRuleInput) (RuleView, error)
}

// Config stores features as the id or key the caller sent (v1 accepts keys); Features
// resolves them, archived ones included, so a read-modify-write round trip keeps the
// rule's scope.
type RuleView struct {
	Rule

	Features []feature.Feature
}
