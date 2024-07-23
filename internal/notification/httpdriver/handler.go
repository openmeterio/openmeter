package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/notification"
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
}

type EventHandler interface {
	ListEvents() ListEventsHandler
	GetEvent() GetEventHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	connector        notification.Connector
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
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
	connector notification.Connector,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		connector:        connector,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}
}
