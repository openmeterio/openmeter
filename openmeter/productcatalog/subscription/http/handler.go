package httpdriver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateSubscription() CreateSubscriptionHandler
	GetSubscription() GetSubscriptionHandler
	EditSubscription() EditSubscriptionHandler
	CancelSubscription() CancelSubscriptionHandler
	ContinueSubscription() ContinueSubscriptionHandler
	MigrateSubscription() MigrateSubscriptionHandler
	ChangeSubscription() ChangeSubscriptionHandler
}

type HandlerConfig struct {
	SubscriptionWorkflowService subscription.WorkflowService
	SubscriptionService         subscription.Service
	PlanSubscriptionService     plansubscription.PlanSubscriptionService
	NamespaceDecoder            namespacedriver.NamespaceDecoder
	Logger                      *slog.Logger
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

func NewHandler(config HandlerConfig, options ...httptransport.HandlerOption) Handler {
	return &handler{
		HandlerConfig: config,
		Options:       options,
	}
}
