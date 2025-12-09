package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers"
	"github.com/openmeterio/openmeter/api/v3/render"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Config struct {
	BaseURL          string
	NamespaceDecoder namespacedriver.NamespaceDecoder
	ErrorHandler     errorsx.Handler

	// services
	CustomerService customer.Service
	MeterService    meter.Service
}

type Server struct {
	*Config

	swagger         *openapi3.T
	customerHandler handlers.CustomerHandler
	meterHandler    handlers.MeterHandler
	middlewares     []api.MiddlewareFunc
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Server)(nil)

func NewServer(config *Config) (*Server, error) {
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

	middlewares := []api.MiddlewareFunc{oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
		// ErrorHandlerWithOpts: func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request, opts oapimiddleware.ErrorHandlerOpts) {
		// 	models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)
		// },
		SilenceServersWarning: true,
		Options: openapi3filter.Options{
			// NoOp authenticationFunc as it's handled in another middleware
			// this is based on `security` property on OpenAPI Spec
			AuthenticationFunc:        openapi3filter.NoopAuthenticationFunc,
			MultiError:                true,
			SkipSettingDefaults:       false,
			ExcludeRequestQueryParams: true,
		},
	})}

	resolveNamespace := func(ctx context.Context) (string, error) {
		ns, ok := config.NamespaceDecoder.GetNamespace(ctx)
		if !ok {
			return "", apierrors.NewInternalError(ctx, errors.New("failed to resolve namespace"))
		}

		return ns, nil
	}

	customerHandler := handlers.NewCustomerHandler(resolveNamespace, config.CustomerService, httptransport.WithErrorHandler(config.ErrorHandler))
	meterHandler := handlers.NewMeterHandler(resolveNamespace, config.MeterService, httptransport.WithErrorHandler(config.ErrorHandler))

	return &Server{
		Config:          config,
		swagger:         swagger,
		middlewares:     middlewares,
		customerHandler: customerHandler,
		meterHandler:    meterHandler,
	}, nil
}

func (s *Server) RegisterRoutes(r chi.Router) {
	r.Route(s.BaseURL, func(r chi.Router) {
		// Serve the OpenAPI spec
		r.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
			_ = render.RenderJSON(w, s.swagger)
		})

		r.Get("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
			_ = render.RenderYAML(w, s.swagger)
		})

		_ = api.HandlerWithOptions(s, api.ChiServerOptions{
			BaseRouter:  r,
			Middlewares: s.middlewares,
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				s.ErrorHandler.HandleContext(r.Context(), err)
			},
		})
	})
}
