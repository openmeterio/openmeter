package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
	TaxCodeResolver             productcatalog.TaxCodeResolver
}

func NewHandler(config HandlerConfig, options ...httptransport.HandlerOption) Handler {
	return &handler{
		HandlerConfig: config,
		Options:       options,
	}
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

// taxCodesForAddons looks up, in one batch, the Tax Codes referenced by the add-ons' rate cards, so
// v1 responses can show their stripe.code.
func (h *handler) taxCodesForAddons(ctx context.Context, namespace string, addons ...subscriptionaddon.SubscriptionAddon) (map[string]*taxcode.TaxCode, error) {
	var rateCards productcatalog.RateCards

	for _, add := range addons {
		for _, rc := range add.RateCards {
			rateCards = append(rateCards, rc.AddonRateCard.RateCard)
		}
	}

	taxCodes, err := h.TaxCodeResolver.BatchResolve(ctx, namespace, rateCards.TaxCodeIDs()...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
