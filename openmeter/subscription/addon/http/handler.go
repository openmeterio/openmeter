package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateSubscriptionAddon() CreateSubscriptionAddonHandler
	ListSubscriptionAddons() ListSubscriptionAddonsHandler
	GetSubscriptionAddon() GetSubscriptionAddonHandler
	UpdateSubscriptionAddon() UpdateSubscriptionAddonHandler
}

type HandlerConfig struct {
	SubscriptionAddonService    subscriptionaddon.Service
	SubscriptionWorkflowService subscriptionworkflow.Service
	SubscriptionService         subscription.Service
	AddonService                addon.Service
	NamespaceDecoder            namespacedriver.NamespaceDecoder
	Logger                      *slog.Logger
}

func (c HandlerConfig) Validate() error {
	var errs []error

	if isNil(c.SubscriptionAddonService) {
		errs = append(errs, errors.New("subscription add-on service is required"))
	}

	if isNil(c.SubscriptionWorkflowService) {
		errs = append(errs, errors.New("subscription workflow service is required"))
	}

	if isNil(c.SubscriptionService) {
		errs = append(errs, errors.New("subscription service is required"))
	}

	if isNil(c.AddonService) {
		errs = append(errs, errors.New("add-on service is required"))
	}

	if isNil(c.NamespaceDecoder) {
		errs = append(errs, errors.New("namespace decoder is required"))
	}

	if isNil(c.Logger) {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

func NewHandler(config HandlerConfig, options ...httptransport.HandlerOption) (Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subscription add-on handler config: %w", err)
	}

	return &handler{
		HandlerConfig: config,
		Options:       options,
	}, nil
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
