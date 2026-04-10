package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	appshandler "github.com/openmeterio/openmeter/api/v3/handlers/apps"
	billingprofileshandler "github.com/openmeterio/openmeter/api/v3/handlers/billingprofiles"
	currencieshandler "github.com/openmeterio/openmeter/api/v3/handlers/currencies"
	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	customersbillinghandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/billing"
	customerscreditshandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/credits"
	customersentitlementhandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/entitlementaccess"
	eventshandler "github.com/openmeterio/openmeter/api/v3/handlers/events"
	featurecosthandler "github.com/openmeterio/openmeter/api/v3/handlers/featurecost"
	featureshandler "github.com/openmeterio/openmeter/api/v3/handlers/features"
	llmcosthandler "github.com/openmeterio/openmeter/api/v3/handlers/llmcost"
	metershandler "github.com/openmeterio/openmeter/api/v3/handlers/meters"
	planshandler "github.com/openmeterio/openmeter/api/v3/handlers/plans"
	subscriptionshandler "github.com/openmeterio/openmeter/api/v3/handlers/subscriptions"
	taxcodeshandler "github.com/openmeterio/openmeter/api/v3/handlers/taxcodes"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
	"github.com/openmeterio/openmeter/api/v3/render"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/server"
)

type Config struct {
	BaseURL             string
	NamespaceDecoder    namespacedriver.NamespaceDecoder
	ErrorHandler        errorsx.Handler
	Middlewares         []server.MiddlewareFunc
	PostAuthMiddlewares []server.MiddlewareFunc
	Credits             config.CreditsConfiguration

	// services
	AppService              app.Service
	BillingService          billing.Service
	LLMCostService          llmcost.Service
	MeterService            meter.ManageService
	StreamingConnector      streaming.Connector
	IngestService           ingest.Service
	CustomerService         customer.Service
	CreditGrantService      creditgrant.Service
	CustomerBalanceFacade   *customerbalance.Facade
	EntitlementService      entitlement.Service
	PlanService             plan.Service
	PlanAddonService        planaddon.Service
	PlanSubscriptionService plansubscription.PlanSubscriptionService
	StripeService           appstripe.Service
	SubscriptionService     subscription.Service
	TaxCodeService          taxcode.Service
	CurrencyService         currencies.CurrencyService
	CostService             cost.Service
	FeatureConnector        feature.FeatureConnector
}

func (c *Config) Validate() error {
	var errs []error

	if c.BaseURL == "" {
		errs = append(errs, errors.New("base URL is required"))
	}

	if c.NamespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}

	if c.ErrorHandler == nil {
		errs = append(errs, errors.New("error handler is required"))
	}

	if c.AppService == nil {
		errs = append(errs, errors.New("app service is required"))
	}

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service is required"))
	}

	if c.MeterService == nil {
		errs = append(errs, errors.New("meter service is required"))
	}

	if c.StreamingConnector == nil {
		errs = append(errs, errors.New("streaming connector is required"))
	}

	if c.IngestService == nil {
		errs = append(errs, errors.New("ingest service is required"))
	}

	if c.CustomerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	if c.EntitlementService == nil {
		errs = append(errs, errors.New("entitlement service is required"))
	}

	if c.PlanService == nil {
		errs = append(errs, errors.New("plan service is required"))
	}

	if c.PlanSubscriptionService == nil {
		errs = append(errs, errors.New("plan subscription service is required"))
	}

	if c.StripeService == nil {
		errs = append(errs, errors.New("stripe service is required"))
	}

	if c.SubscriptionService == nil {
		errs = append(errs, errors.New("subscription service is required"))
	}

	if c.TaxCodeService == nil {
		errs = append(errs, errors.New("tax code service is required"))
	}

	if c.CurrencyService == nil {
		errs = append(errs, errors.New("currency service is required"))
	}

	if c.FeatureConnector == nil {
		errs = append(errs, errors.New("feature connector is required"))
	}

	return errors.Join(errs...)
}

type Server struct {
	*Config

	swagger *openapi3.T

	// handlers
	appsHandler                 appshandler.Handler
	eventsHandler               eventshandler.Handler
	llmcostHandler              llmcosthandler.Handler
	customersHandler            customershandler.Handler
	customersBillingHandler     customersbillinghandler.Handler
	customersCreditsHandler     customerscreditshandler.Handler
	customersEntitlementHandler customersentitlementhandler.Handler
	metersHandler               metershandler.Handler
	subscriptionsHandler        subscriptionshandler.Handler
	billingProfilesHandler      billingprofileshandler.Handler
	plansHandler                planshandler.Handler
	taxcodesHandler             taxcodeshandler.Handler
	currenciesHandler           currencieshandler.Handler
	featuresHandler             featureshandler.Handler
	featureCostHandler          featurecosthandler.Handler
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Server)(nil)

func NewServer(config *Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid v3 server config: %w", err)
	}

	// Get the OpenAPI spec
	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("failed to get swagger", "error", err)
		return nil, err
	}

	// Set the server URL to the base URL to make validation work on the base URL
	swagger.Servers = []*openapi3.Server{
		{
			URL: config.BaseURL,
		},
	}

	resolveNamespace := func(ctx context.Context) (string, error) {
		ns, ok := config.NamespaceDecoder.GetNamespace(ctx)
		if !ok {
			return "", apierrors.NewInternalError(ctx, errors.New("failed to resolve namespace"))
		}

		return ns, nil
	}

	appsHandler := appshandler.New(resolveNamespace, config.AppService, httptransport.WithErrorHandler(config.ErrorHandler))
	eventsHandler := eventshandler.New(resolveNamespace, config.IngestService, httptransport.WithErrorHandler(config.ErrorHandler))
	customersHandler := customershandler.New(resolveNamespace, config.CustomerService, httptransport.WithErrorHandler(config.ErrorHandler))
	customersBillingHandler := customersbillinghandler.New(resolveNamespace, config.BillingService, config.CustomerService, config.StripeService, httptransport.WithErrorHandler(config.ErrorHandler))
	var customersCreditsHandler customerscreditshandler.Handler
	if config.CustomerBalanceFacade != nil && config.Credits.Enabled {
		customersCreditsHandler = customerscreditshandler.New(resolveNamespace, config.CustomerService, config.CustomerBalanceFacade, config.CreditGrantService, httptransport.WithErrorHandler(config.ErrorHandler))
	}
	customersEntitlementHandler := customersentitlementhandler.New(resolveNamespace, config.CustomerService, config.EntitlementService, httptransport.WithErrorHandler(config.ErrorHandler))
	metersHandler := metershandler.New(resolveNamespace, config.MeterService, config.StreamingConnector, config.CustomerService, httptransport.WithErrorHandler(config.ErrorHandler))
	subscriptionsHandler := subscriptionshandler.New(resolveNamespace, config.CustomerService, config.PlanService, config.PlanSubscriptionService, config.SubscriptionService, httptransport.WithErrorHandler(config.ErrorHandler))
	billingProfilesHandler := billingprofileshandler.New(resolveNamespace, config.BillingService, httptransport.WithErrorHandler(config.ErrorHandler))
	plansHandler := planshandler.New(resolveNamespace, config.PlanService, config.PlanAddonService, httptransport.WithErrorHandler(config.ErrorHandler))
	taxcodesHandler := taxcodeshandler.New(resolveNamespace, config.TaxCodeService, httptransport.WithErrorHandler(config.ErrorHandler))
	currenciesHandler := currencieshandler.New(config.NamespaceDecoder, config.CurrencyService, httptransport.WithErrorHandler(config.ErrorHandler))

	featuresH := featureshandler.New(resolveNamespace, config.FeatureConnector, config.MeterService, config.LLMCostService, httptransport.WithErrorHandler(config.ErrorHandler))

	var llmcostH llmcosthandler.Handler
	if config.LLMCostService != nil {
		llmcostH = llmcosthandler.New(resolveNamespace, config.LLMCostService, httptransport.WithErrorHandler(config.ErrorHandler))
	}

	var featureCostH featurecosthandler.Handler
	if config.CostService != nil && config.FeatureConnector != nil {
		featureCostH = featurecosthandler.New(resolveNamespace, config.CostService, config.FeatureConnector, config.MeterService, config.CustomerService, httptransport.WithErrorHandler(config.ErrorHandler))
	}

	return &Server{
		Config:                      config,
		swagger:                     swagger,
		appsHandler:                 appsHandler,
		eventsHandler:               eventsHandler,
		llmcostHandler:              llmcostH,
		customersHandler:            customersHandler,
		customersBillingHandler:     customersBillingHandler,
		customersCreditsHandler:     customersCreditsHandler,
		customersEntitlementHandler: customersEntitlementHandler,
		metersHandler:               metersHandler,
		subscriptionsHandler:        subscriptionsHandler,
		billingProfilesHandler:      billingProfilesHandler,
		plansHandler:                plansHandler,
		taxcodesHandler:             taxcodesHandler,
		currenciesHandler:           currenciesHandler,
		featuresHandler:             featuresH,
		featureCostHandler:          featureCostH,
	}, nil
}

func (s *Server) RegisterRoutes(r chi.Router) error {
	validationRouter, err := oasmiddleware.NewValidationRouter(
		context.Background(),
		s.swagger,
		&oasmiddleware.ValidationRouterOpts{
			DeleteServers: true,
			ServerPrefix:  s.BaseURL,
		},
	)
	if err != nil {
		return fmt.Errorf("create validation router: %w", err)
	}

	validationMiddleware := oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{
		RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook,
		RouteValidationErrorHook: func(err error, w http.ResponseWriter, r *http.Request) bool {
			return oasmiddleware.OasValidationErrorHook(r.Context(), err, w, r)
		},
		FilterOptions: &openapi3filter.Options{
			// No-op auth: auth is handled by other middleware.
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			MultiError:         true,
		},
	})

	r.Route(s.BaseURL, func(r chi.Router) {
		for _, mw := range s.Middlewares {
			r.Use(mw)
		}
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			apierrors.
				NewNotFoundError(r.Context(), errors.New("route not found"), "route").
				HandleAPIError(w, r)
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			apierrors.
				NewMethodNotAllowedError(r.Context()).
				HandleAPIError(w, r)
		})

		// Serve the OpenAPI spec
		r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
			_ = render.RenderJSON(w, s.swagger)
		})

		r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
			_ = render.RenderYAML(w, s.swagger)
		})

		middlewares := []api.MiddlewareFunc{
			validationMiddleware,
		}

		postAuthMiddlewares := lo.Map(s.PostAuthMiddlewares, func(mwf server.MiddlewareFunc, _ int) api.MiddlewareFunc {
			return api.MiddlewareFunc(mwf)
		})

		middlewares = append(middlewares, postAuthMiddlewares...)

		_ = api.HandlerWithOptions(s, api.ChiServerOptions{
			BaseRouter:       r,
			Middlewares:      middlewares,
			ErrorHandlerFunc: apierrors.NewV3ErrorHandlerFunc(s.ErrorHandler),
		})
	})

	return nil
}
