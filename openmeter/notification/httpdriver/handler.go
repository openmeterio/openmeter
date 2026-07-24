package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/internal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ChannelHandler
	RuleHandler
	EventHandler
}

type ChannelHandler interface {
	ListChannels() ListChannelsHandler
	CreateChannel() CreateChannelHandler
	DeleteChannel() DeleteChannelHandler
	GetChannel() GetChannelHandler
	UpdateChannel() UpdateChannelHandler
}

type RuleHandler interface {
	ListRules() ListRulesHandler
	CreateRule() CreateRuleHandler
	DeleteRule() DeleteRuleHandler
	GetRule() GetRuleHandler
	UpdateRule() UpdateRuleHandler
	TestRule() TestRuleHandler
}

type EventHandler interface {
	ListEvents() ListEventsHandler
	GetEvent() GetEventHandler
	ResendEvent() ResendEventHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service            notification.Service
	testEventGenerator *internal.TestEventGenerator
	namespaceDecoder   namespacedriver.NamespaceDecoder
	options            []httptransport.HandlerOption
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func New(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	service notification.Service,
	billingService billing.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if service == nil {
		errs = append(errs, errors.New("notification service is required"))
	}
	if billingService == nil {
		errs = append(errs, errors.New("billing service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid notification handler config: %w", err)
	}

	return &handler{
		service:            service,
		testEventGenerator: internal.NewTestEventGenerator(billingService),
		namespaceDecoder:   namespaceDecoder,
		options:            options,
	}, nil
}
