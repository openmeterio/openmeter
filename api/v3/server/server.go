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
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	customershandler "github.com/openmeterio/openmeter/api/v3/handlers/customers"
	customersentitlementhandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/entitlementaccess"
	eventshandler "github.com/openmeterio/openmeter/api/v3/handlers/events"
	metershandler "github.com/openmeterio/openmeter/api/v3/handlers/meters"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
	"github.com/openmeterio/openmeter/api/v3/render"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Config struct {
	BaseURL          string
	NamespaceDecoder namespacedriver.NamespaceDecoder
	ErrorHandler     errorsx.Handler
	Middlewares      []func(http.Handler) http.Handler

	// services
	MeterService       meter.ManageService
	IngestService      ingest.Service
	CustomerService    customer.Service
	EntitlementService entitlement.Service
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

	if c.MeterService == nil {
		errs = append(errs, errors.New("meter service is required"))
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

	return errors.Join(errs...)
}

type Server struct {
	*Config

	swagger *openapi3.T

	// handlers
	eventsHandler               eventshandler.Handler
	customersHandler            customershandler.Handler
	customersEntitlementHandler customersentitlementhandler.Handler
	metersHandler               metershandler.Handler
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

	eventsHandler := eventshandler.New(resolveNamespace, config.IngestService, httptransport.WithErrorHandler(config.ErrorHandler))
	customersHandler := customershandler.New(resolveNamespace, config.CustomerService, httptransport.WithErrorHandler(config.ErrorHandler))
	customersEntitlementHandler := customersentitlementhandler.New(resolveNamespace, config.CustomerService, config.EntitlementService, httptransport.WithErrorHandler(config.ErrorHandler))
	metersHandler := metershandler.New(resolveNamespace, config.MeterService, httptransport.WithErrorHandler(config.ErrorHandler))

	return &Server{
		Config:                      config,
		swagger:                     swagger,
		eventsHandler:               eventsHandler,
		customersHandler:            customersHandler,
		customersEntitlementHandler: customersEntitlementHandler,
		metersHandler:               metersHandler,
	}, nil
}

func (s *Server) RegisterRoutes(r chi.Router) {
	r.Route(s.BaseURL, func(r chi.Router) {
		r.Use(s.Middlewares...)
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

		_ = api.HandlerWithOptions(s, api.ChiServerOptions{
			BaseRouter: r,
			Middlewares: []api.MiddlewareFunc{oapimiddleware.OapiRequestValidatorWithOptions(s.swagger, &oapimiddleware.Options{
				ErrorHandlerWithOpts: func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request, opts oapimiddleware.ErrorHandlerOpts) {
					oasmiddleware.OasValidationErrorHook(ctx, err, w, r)
				},
				SilenceServersWarning: true,
				Options: openapi3filter.Options{
					AuthenticationFunc:        openapi3filter.NoopAuthenticationFunc,
					SkipSettingDefaults:       false,
					MultiError:                true,
					ExcludeRequestQueryParams: true,
				},
			})},
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				s.ErrorHandler.HandleContext(r.Context(), err)
			},
		})
	})
}
