package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	appconfig "github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
	TaxCodeResolver             productcatalog.TransactionalTaxCodeResolver
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

// taxCodesForView looks up, in one batch, the Tax Codes referenced by the view's items, so v1
// responses can show their stripe.code.
func (h *handler) taxCodesForView(ctx context.Context, view subscription.SubscriptionView) (map[string]*taxcode.TaxCode, error) {
	var rateCards productcatalog.RateCards

	for _, phase := range view.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				rateCards = append(rateCards, item.SubscriptionItem.RateCard)
			}
		}
	}

	taxCodes, err := h.TaxCodeResolver.BatchResolve(ctx, view.Subscription.Namespace, rateCards.TaxCodeIDs()...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
