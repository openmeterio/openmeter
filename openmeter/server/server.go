package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/openmeterio/openmeter/api"
	v3server "github.com/openmeterio/openmeter/api/v3/server"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/portal/authenticator"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/server"
)

type Server struct {
	chi.Router
}

type ServerLogger struct{}

type MiddlewareManager interface {
	Use(middlewares ...func(http.Handler) http.Handler)
}

type MiddlewareHook func(m MiddlewareManager)

type RouteManager interface {
	Mount(pattern string, h http.Handler)
	Handle(pattern string, h http.Handler)
	HandleFunc(pattern string, h http.HandlerFunc)
	Method(method, pattern string, h http.Handler)
	MethodFunc(method, pattern string, h http.HandlerFunc)
	Connect(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Get(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
	Options(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Trace(pattern string, h http.HandlerFunc)
}

type RouteHook func(r RouteManager)

type RouterHooks struct {
	Middlewares []MiddlewareHook
	Routes      []RouteHook
}

type PostAuthMiddlewares []api.MiddlewareFunc

type Config struct {
	RouterConfig        router.Config
	RouterHooks         RouterHooks
	PostAuthMiddlewares PostAuthMiddlewares
}

func NewServer(config *Config) (*Server, error) {
	// Get the OpenAPI spec
	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("failed to get swagger", "error", err)
		return nil, err
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	impl, err := router.NewRouter(config.RouterConfig)
	if err != nil {
		slog.Error("failed to create API", "error", err)
		return nil, err
	}

	r := chi.NewRouter()

	v3API, err := v3server.NewServer(&v3server.Config{
		BaseURL:          "/api/v3",
		NamespaceDecoder: namespacedriver.StaticNamespaceDecoder(config.RouterConfig.NamespaceManager.GetDefaultNamespace()),
		ErrorHandler:     config.RouterConfig.ErrorHandler,
		IngestService:    config.RouterConfig.IngestService,
		CustomerService:  config.RouterConfig.Customer,
		MeterService:     config.RouterConfig.MeterManageService,
		Middlewares: []func(http.Handler) http.Handler{
			middleware.RealIP,
			middleware.RequestID,
			func(h http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := r.Context()
					ctx = contextx.WithAttrs(ctx, server.GetRequestAttributes(r))

					h.ServeHTTP(w, r.WithContext(ctx))
				})
			},
			server.NewRequestLoggerMiddleware(slog.Default().Handler()),
			middleware.Recoverer,
		},
	})
	if err != nil {
		slog.Error("failed to create v3 API", "error", err)
		return nil, err
	}

	r.Group(func(r chi.Router) {
		v3API.RegisterRoutes(r)
	})

	r.Group(func(r chi.Router) {
		// Apply middlewares
		for _, middlewareHook := range config.RouterHooks.Middlewares {
			middlewareHook(r)
		}

		r.Use(middleware.RealIP)
		r.Use(middleware.RequestID)
		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				ctx = contextx.WithAttrs(ctx, server.GetRequestAttributes(r))

				h.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		r.Use(server.NewRequestLoggerMiddleware(slog.Default().Handler()))
		r.Use(middleware.Recoverer)
		if config.RouterConfig.PortalCORSEnabled {
			// Enable CORS for portal requests
			r.Use(corsHandler(corsOptions{
				AllowedPaths: []string{"/api/v1/portal/meters"},
				Options: cors.Options{
					AllowOriginFunc: func(r *http.Request, origin string) bool {
						return true
					},
					AllowedMethods:   []string{http.MethodGet, http.MethodOptions},
					AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
					AllowCredentials: true,
					MaxAge:           1728000,
				},
			}))
		}
		r.Use(render.SetContentType(render.ContentTypeJSON))
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			models.NewStatusProblem(r.Context(), nil, http.StatusNotFound).Respond(w)
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			models.NewStatusProblem(r.Context(), nil, http.StatusMethodNotAllowed).Respond(w)
		})

		// Serve the OpenAPI spec
		r.Get("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, swagger)
		})

		// Apply route handlers
		for _, routeHook := range config.RouterHooks.Routes {
			routeHook(r)
		}

		middlewares := []api.MiddlewareFunc{
			authenticator.NewAuthenticator(config.RouterConfig.Portal, config.RouterConfig.ErrorHandler).NewAuthenticatorMiddlewareFunc(swagger),
			oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
				ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
					models.NewStatusProblem(context.Background(), errors.New(message), statusCode).Respond(w)
				},
				Options: openapi3filter.Options{
					// Unfortunately, the OpenAPI 3 filter library doesn't support context changes
					AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
					SkipSettingDefaults: true,

					// Excluding read-only validation because required and readOnly fields in our Go models are translated to non-nil fields, leading to a zero-value being passed to the API
					// The OpenAPI spec says read-only fields SHOULD NOT be sent in requests, so technically it should be fine, hence disabling validation for now to make our life easier
					ExcludeReadOnlyValidations: true,
				},
			}),
		}

		middlewares = append(middlewares, config.PostAuthMiddlewares...)

		// Use validator middleware to check requests against the OpenAPI schema
		_ = api.HandlerWithOptions(impl, api.ChiServerOptions{
			BaseRouter:  r,
			Middlewares: middlewares,
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				config.RouterConfig.ErrorHandler.HandleContext(r.Context(), err)
				errorHandlerReply(w, r, err)
			},
		})
	})

	return &Server{
		Router: r,
	}, nil
}

// errorHandlerReply handles errors returned by the OpenAPI layer.
func errorHandlerReply(w http.ResponseWriter, r *http.Request, err error) {
	switch e := err.(type) {
	case *api.UnescapedCookieParamError:
		err := fmt.Errorf("unescaped cookie param %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	case *api.UnmarshalingParamError:
		err := fmt.Errorf("unmarshaling param %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	case *api.RequiredParamError:
		err := fmt.Errorf("required param missing %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	case *api.RequiredHeaderError:
		err := fmt.Errorf("required header missing %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	case *api.InvalidParamFormatError:
		err := fmt.Errorf("invalid param format %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	case *api.TooManyValuesForParamError:
		err := fmt.Errorf("too many values for param %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	default:
		err := fmt.Errorf("unhandled server error: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)
	}
}
