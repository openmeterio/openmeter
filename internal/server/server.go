package server

import (
	"context"
	"errors"
	"net/http"

	oapimiddleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/api"
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
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		models.NewStatusProblem(r.Context(), nil, http.StatusNotFound).Respond(w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		models.NewStatusProblem(r.Context(), nil, http.StatusMethodNotAllowed).Respond(w, r)
	})

	// Serve the OpenAPI spec
	r.Get("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, swagger)
	})

	// Use validator middleware to check requests against the OpenAPI schema
	_ = api.HandlerWithOptions(impl, api.ChiServerOptions{
		BaseRouter: r,
		Middlewares: []api.MiddlewareFunc{
			oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
				ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
					models.NewStatusProblem(context.Background(), errors.New(message), statusCode).Respond(w, nil)
				},
				Options: openapi3filter.Options{
					AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
					SkipSettingDefaults: true,
				},
			}),
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		},
	})

	return &Server{
		Router: r,
	}, nil
}
