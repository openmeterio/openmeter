package rules

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/testevent"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListNotificationRules() ListNotificationRulesHandler
	CreateNotificationRule() CreateNotificationRuleHandler
	GetNotificationRule() GetNotificationRuleHandler
	UpdateNotificationRule() UpdateNotificationRuleHandler
	DeleteNotificationRule() DeleteNotificationRuleHandler
	TestNotificationRule() TestNotificationRuleHandler
}

type handler struct {
	resolveNamespace   func(ctx context.Context) (string, error)
	service            notification.Service
	testEventGenerator *testevent.Generator
	options            []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service notification.Service,
	billingService billing.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:   resolveNamespace,
		service:            service,
		testEventGenerator: testevent.NewGenerator(billingService),
		options:            options,
	}
}
