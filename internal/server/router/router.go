package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	creditdriver "github.com/openmeterio/openmeter/internal/credit/driver"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/debug"
	debug_httpdriver "github.com/openmeterio/openmeter/internal/debug/httpdriver"
	"github.com/openmeterio/openmeter/internal/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/internal/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/notification"
	notificationhttpdriver "github.com/openmeterio/openmeter/internal/notification/httpdriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalog_httpdriver "github.com/openmeterio/openmeter/internal/productcatalog/driver"
	"github.com/openmeterio/openmeter/internal/server/authenticator"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

func init() {
	// See https://github.com/getkin/kin-openapi/issues/640
	openapi3filter.RegisterBodyDecoder("application/cloudevents+json", jsonBodyDecoder)
	openapi3filter.RegisterBodyDecoder("application/cloudevents-batch+json", jsonBodyDecoder)
}

func jsonBodyDecoder(body io.Reader, header http.Header, schema *openapi3.SchemaRef, encFn openapi3filter.EncodingFn) (interface{}, error) {
	var value interface{}
	if err := json.NewDecoder(body).Decode(&value); err != nil {
		return nil, &openapi3filter.ParseError{Kind: openapi3filter.KindInvalidFormat, Cause: err}
	}
	return value, nil
}

type IngestHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string)
}

type Config struct {
	NamespaceManager    *namespace.Manager
	StreamingConnector  streaming.Connector
	IngestHandler       http.Handler
	Meters              meter.Repository
	PortalCORSEnabled   bool
	PortalTokenStrategy *authenticator.PortalTokenStrategy
	ErrorHandler        errorsx.Handler

	// deps
	DebugConnector              debug.DebugConnector
	FeatureConnector            productcatalog.FeatureConnector
	EntitlementConnector        entitlement.Connector
	EntitlementBalanceConnector meteredentitlement.Connector
	GrantConnector              credit.GrantConnector
	GrantRepo                   grant.Repo
	NotificationConnector       notification.Connector

	// FIXME: implement generic module management, loading, etc...
	EntitlementsEnabled bool
}

func (c Config) Validate() error {
	if c.NamespaceManager == nil {
		return errors.New("namespace manager is required")
	}

	if c.ErrorHandler == nil {
		return errors.New("error handler is required")
	}

	if c.IngestHandler == nil {
		return errors.New("ingest handler is required")
	}

	// Validate repositories
	if c.Meters == nil {
		return errors.New("meters repository is required")
	}

	// Validate connectors
	if c.StreamingConnector == nil {
		return errors.New("streaming connector is required")
	}

	if c.DebugConnector == nil {
		return errors.New("debug connector is required")
	}

	if c.EntitlementsEnabled {
		if c.FeatureConnector == nil {
			return errors.New("feature connector is required")
		}

		if c.EntitlementConnector == nil {
			return errors.New("entitlement connector is required")
		}

		if c.EntitlementBalanceConnector == nil {
			return errors.New("entitlement balance connector is required")
		}

		if c.GrantConnector == nil {
			return errors.New("grant connector is required")
		}
	}

	if c.NotificationConnector == nil {
		return errors.New("notification connector is required")
	}

	return nil
}

type Router struct {
	config Config

	featureHandler            productcatalog_httpdriver.FeatureHandler
	creditHandler             creditdriver.GrantHandler
	debugHandler              debug_httpdriver.DebugHandler
	entitlementHandler        entitlementdriver.EntitlementHandler
	meteredEntitlementHandler entitlementdriver.MeteredEntitlementHandler
	notificationHandler       notificationhttpdriver.Handler
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Router)(nil)

func NewRouter(config Config) (*Router, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid router config: %w", err)
	}

	router := &Router{
		config: config,
	}

	staticNamespaceDecoder := namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace())

	router.debugHandler = debug_httpdriver.NewDebugHandler(
		staticNamespaceDecoder,
		config.DebugConnector,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	if config.EntitlementsEnabled {
		router.featureHandler = productcatalog_httpdriver.NewFeatureHandler(
			config.FeatureConnector,
			staticNamespaceDecoder,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		router.entitlementHandler = entitlementdriver.NewEntitlementHandler(
			config.EntitlementConnector,
			staticNamespaceDecoder,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		router.meteredEntitlementHandler = entitlementdriver.NewMeteredEntitlementHandler(
			config.EntitlementConnector,
			config.EntitlementBalanceConnector,
			staticNamespaceDecoder,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		router.creditHandler = creditdriver.NewGrantHandler(
			staticNamespaceDecoder,
			config.GrantConnector,
			config.GrantRepo,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)
	}

	router.notificationHandler = notificationhttpdriver.New(
		staticNamespaceDecoder,
		config.NotificationConnector,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	return router, nil
}
