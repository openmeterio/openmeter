package router

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	credit_httpdriver "github.com/openmeterio/openmeter/internal/credit/httpdriver"
	"github.com/openmeterio/openmeter/internal/entitlement"
	entitlement_httpdriver "github.com/openmeterio/openmeter/internal/entitlement/httpdriver"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalog_httpdriver "github.com/openmeterio/openmeter/internal/productcatalog/httpdriver"
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
	FeatureConnector            productcatalog.FeatureConnector
	EntitlementConnector        entitlement.EntitlementConnector
	EntitlementBalanceConnector entitlement.EntitlementBalanceConnector
	GrantConnector              credit.GrantConnector

	// FIXME: implement generic module management, loading, etc...
	EntitlementsEnabled bool
}

type Router struct {
	config Config

	featureHandler            productcatalog_httpdriver.FeatureHandler
	creditHandler             credit_httpdriver.GrantHandler
	entitlementHandler        entitlement_httpdriver.EntitlementHandler
	meteredEntitlementHandler entitlement_httpdriver.MeteredEntitlementHandler
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Router)(nil)

func NewRouter(config Config) (*Router, error) {
	router := &Router{
		config: config,
	}

	if config.EntitlementsEnabled {
		router.featureHandler = productcatalog_httpdriver.NewFeatureHandler(
			config.FeatureConnector,
			namespacedriver.StaticNamespaceDecoder("default"),
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		router.entitlementHandler = entitlement_httpdriver.NewEntitlementHandler(
			config.EntitlementConnector,
			namespacedriver.StaticNamespaceDecoder("default"),
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		router.meteredEntitlementHandler = entitlement_httpdriver.NewMeteredEntitlementHandler(
			config.EntitlementConnector,
			config.EntitlementBalanceConnector,
			namespacedriver.StaticNamespaceDecoder("default"),
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		router.creditHandler = credit_httpdriver.NewGrantHandler(
			namespacedriver.StaticNamespaceDecoder("default"),
			config.GrantConnector,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)
	}

	return router, nil
}
