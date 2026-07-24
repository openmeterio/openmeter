package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	appconfig "github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateSubscription() CreateSubscriptionHandler
	GetSubscription() GetSubscriptionHandler
	EditSubscription() EditSubscriptionHandler
	CancelSubscription() CancelSubscriptionHandler
	ContinueSubscription() ContinueSubscriptionHandler
	RestoreSubscription() RestoreSubscriptionHandler
	MigrateSubscription() MigrateSubscriptionHandler
	ChangeSubscription() ChangeSubscriptionHandler
	DeleteSubscription() DeleteSubscriptionHandler
	ListCustomerSubscriptions() ListCustomerSubscriptionsHandler
}

type HandlerConfig struct {
	SubscriptionWorkflowService subscriptionworkflow.Service
	SubscriptionService         subscription.Service
	CustomerService             customer.Service
	PlanSubscriptionService     plansubscription.PlanSubscriptionService
	NamespaceDecoder            namespacedriver.NamespaceDecoder
	Logger                      *slog.Logger
	Credits                     appconfig.CreditsConfiguration
}

func (c HandlerConfig) Validate() error {
	var errs []error

	if isNil(c.SubscriptionWorkflowService) {
		errs = append(errs, errors.New("subscription workflow service is required"))
	}

	if isNil(c.SubscriptionService) {
		errs = append(errs, errors.New("subscription service is required"))
	}

	if isNil(c.CustomerService) {
		errs = append(errs, errors.New("customer service is required"))
	}

	if isNil(c.PlanSubscriptionService) {
		errs = append(errs, errors.New("plan subscription service is required"))
	}

	if isNil(c.NamespaceDecoder) {
		errs = append(errs, errors.New("namespace decoder is required"))
	}

	if isNil(c.Logger) {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

type handler struct {
	HandlerConfig
	Options []httptransport.HandlerOption
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.NamespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func NewHandler(config HandlerConfig, options ...httptransport.HandlerOption) (Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subscription handler config: %w", err)
	}

	return &handler{
		HandlerConfig: config,
		Options:       options,
	}, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
