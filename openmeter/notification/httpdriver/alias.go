package httpdriver

import "github.com/openmeterio/openmeter/internal/notification/httpdriver"

type (
	Handler              = httpdriver.Handler
	ListChannelsHandler  = httpdriver.ListChannelsHandler
	CreateChannelHandler = httpdriver.CreateChannelHandler
	DeleteChannelHandler = httpdriver.DeleteChannelHandler
	GetChannelHandler    = httpdriver.GetChannelHandler
	UpdateChannelHandler = httpdriver.UpdateChannelHandler
	ListRulesHandler     = httpdriver.ListRulesHandler
	CreateRuleHandler    = httpdriver.CreateRuleHandler
	DeleteRuleHandler    = httpdriver.DeleteRuleHandler
	GetRuleHandler       = httpdriver.GetRuleHandler
	UpdateRuleHandler    = httpdriver.UpdateRuleHandler
	ListEventsHandler    = httpdriver.ListEventsHandler
	GetEventHandler      = httpdriver.GetEventHandler
)

type (
	ListChannelsRequest   = httpdriver.ListChannelsRequest
	ListChannelsResponse  = httpdriver.ListChannelsResponse
	CreateChannelRequest  = httpdriver.CreateChannelRequest
	CreateChannelResponse = httpdriver.CreateChannelResponse
	DeleteChannelRequest  = httpdriver.DeleteChannelRequest
	DeleteChannelResponse = httpdriver.DeleteChannelResponse
	GetChannelRequest     = httpdriver.GetChannelRequest
	GetChannelResponse    = httpdriver.GetChannelResponse
	UpdateChannelRequest  = httpdriver.UpdateChannelRequest
	UpdateChannelResponse = httpdriver.UpdateChannelResponse
)

type (
	ListRulesRequest   = httpdriver.ListRulesRequest
	ListRulesResponse  = httpdriver.ListRulesResponse
	CreateRuleRequest  = httpdriver.CreateRuleRequest
	CreateRuleResponse = httpdriver.CreateRuleResponse
	DeleteRuleRequest  = httpdriver.DeleteRuleRequest
	DeleteRuleResponse = httpdriver.DeleteRuleResponse
	GetRuleRequest     = httpdriver.GetRuleRequest
	GetRuleResponse    = httpdriver.GetRuleResponse
	UpdateRuleRequest  = httpdriver.UpdateRuleRequest
	UpdateRuleResponse = httpdriver.UpdateRuleResponse
)

type (
	ListEventsRequest  = httpdriver.ListEventsRequest
	ListEventsResponse = httpdriver.ListEventsResponse
	GetEventRequest    = httpdriver.GetEventRequest
	GetEventResponse   = httpdriver.GetEventResponse
)

var NewHandler = httpdriver.New
