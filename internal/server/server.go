package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/server/authenticator"
	"github.com/openmeterio/openmeter/internal/server/router"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Server struct {
	chi.Router
}

type ServerLogger struct{}

type Config struct {
	RouterConfig router.Config
	RouterHook   func(r chi.Router)
}

func NewServer(config *Config) (*Server, error) {
	defaultLogger := slog.Default()

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

	if config.RouterHook != nil {
		config.RouterHook(r)
	}

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(NewStructuredLogger(slog.Default().Handler(), nil))
	r.Use(middleware.Recoverer)
	if config.RouterConfig.PortalCORSEnabled && config.RouterConfig.PortalTokenStrategy != nil {
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
		models.NewStatusProblem(r.Context(), nil, http.StatusNotFound).Respond(defaultLogger, w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		models.NewStatusProblem(r.Context(), nil, http.StatusMethodNotAllowed).Respond(defaultLogger, w, r)
	})

	// Serve the OpenAPI spec
	r.Get("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, swagger)
	})

	// Use validator middleware to check requests against the OpenAPI schema
	_ = api.HandlerWithOptions(impl, api.ChiServerOptions{
		BaseRouter: r,
		Middlewares: []api.MiddlewareFunc{
			authenticator.NewAuthenticator(config.RouterConfig.PortalTokenStrategy).NewAuthenticatorMiddlewareFunc(swagger),
			oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
				ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
					models.NewStatusProblem(context.Background(), errors.New(message), statusCode).Respond(defaultLogger, w, nil)
				},
				Options: openapi3filter.Options{
					// Unfortunately, the OpenAPI 3 filter library doesn't support context changes
					AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
					SkipSettingDefaults: true,
				},
			}),
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(defaultLogger, w, r)
		},
	})

	return &Server{
		Router: r,
	}, nil
}
