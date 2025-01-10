package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	apphttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/credit"
	creditdriver "github.com/openmeterio/openmeter/openmeter/credit/driver"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/debug"
	debug_httpdriver "github.com/openmeterio/openmeter/openmeter/debug/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationhttpdriver "github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	productcatalog_httpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/driver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	plan "github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/server/authenticator"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
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
	Logger              *slog.Logger

	// deps
	App                         app.Service
	AppStripe                   appstripe.Service
	Customer                    customer.Service
	Billing                     billing.Service
	Plan                        plan.Service
	SubscriptionService         subscription.Service
	SubscriptionWorkflowService subscription.WorkflowService
	PlanSubscriptionService     plansubscription.PlanSubscriptionService
	DebugConnector              debug.DebugConnector
	FeatureConnector            feature.FeatureConnector
	EntitlementConnector        entitlement.Connector
	EntitlementBalanceConnector meteredentitlement.Connector
	GrantConnector              credit.GrantConnector
	GrantRepo                   grant.Repo
	Notification                notification.Service

	// FIXME: implement generic module management, loading, etc...
	BillingEnabled        bool
	ProductCatalogEnabled bool
	AppsEnabled           bool
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

	if c.Notification == nil {
		return errors.New("notification service is required")
	}

	if c.AppsEnabled {
		if c.App == nil {
			return errors.New("app service is required")
		}

		if c.AppStripe == nil {
			return errors.New("app stripe service is required")
		}
	}

	if c.BillingEnabled || c.AppsEnabled {
		if c.Customer == nil {
			return errors.New("customer service is required")
		}
	}

	if c.BillingEnabled && c.Billing == nil {
		return errors.New("billing service is required")
	}

	return nil
}

type Router struct {
	config Config

	appHandler                apphttpdriver.Handler
	appStripeHandler          appstripehttpdriver.AppStripeHandler
	billingHandler            billinghttpdriver.Handler
	featureHandler            productcatalog_httpdriver.FeatureHandler
	planHandler               planhttpdriver.Handler
	subscriptionHandler       subscriptionhttpdriver.Handler
	creditHandler             creditdriver.GrantHandler
	debugHandler              debug_httpdriver.DebugHandler
	customerHandler           customerhttpdriver.CustomerHandler
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

	router.notificationHandler = notificationhttpdriver.New(
		staticNamespaceDecoder,
		config.Notification,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	// Customer
	router.customerHandler = customerhttpdriver.New(
		staticNamespaceDecoder,
		config.Customer,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	// App
	router.appHandler = apphttpdriver.New(
		config.Logger,
		staticNamespaceDecoder,
		config.App,
		config.Billing,
		config.AppStripe,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	// App Stripe
	router.appStripeHandler = appstripehttpdriver.New(
		staticNamespaceDecoder,
		config.AppStripe,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	if config.BillingEnabled {
		router.billingHandler = billinghttpdriver.New(
			config.Logger,
			staticNamespaceDecoder,
			config.Billing,
			config.AppStripe,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)
	}

	if config.ProductCatalogEnabled {
		if config.Plan == nil {
			return nil, errors.New("plan service is required when productcatalog is enabled")
		}

		router.planHandler = planhttpdriver.New(
			staticNamespaceDecoder,
			config.Plan,
			httptransport.WithErrorHandler(config.ErrorHandler),
		)

		if config.SubscriptionService == nil || config.SubscriptionWorkflowService == nil || config.PlanSubscriptionService == nil {
			return nil, errors.New("subscription services are required when productcatalog is enabled")
		}

		if config.Logger == nil {
			return nil, errors.New("logger is required when productcatalog is enabled")
		}

		router.subscriptionHandler = subscriptionhttpdriver.NewHandler(
			subscriptionhttpdriver.HandlerConfig{
				SubscriptionWorkflowService: config.SubscriptionWorkflowService,
				SubscriptionService:         config.SubscriptionService,
				PlanSubscriptionService:     config.PlanSubscriptionService,
				NamespaceDecoder:            staticNamespaceDecoder,
				Logger:                      config.Logger,
			},
			httptransport.WithErrorHandler(config.ErrorHandler),
		)
	}

	return router, nil
}
