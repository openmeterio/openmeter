package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/portal/authenticator"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Server struct {
	chi.Router
}

type ServerLogger struct{}

type Config struct {
	RouterConfig router.Config
	RouterHooks  RouterHookManager
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

	if config.RouterConfig.PortalCORSEnabled {
		config.RouterHooks.RegisterMiddleware(101, corsHandler(corsOptions{
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

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		models.NewStatusProblem(r.Context(), nil, http.StatusNotFound).Respond(w)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		models.NewStatusProblem(r.Context(), nil, http.StatusMethodNotAllowed).Respond(w)
	})

	config.RouterHooks.RegisterMiddleware(200, authenticator.NewAuthenticator(config.RouterConfig.Portal, config.RouterConfig.ErrorHandler).NewAuthenticatorMiddlewareFunc(swagger))
	config.RouterHooks.RegisterMiddleware(200, oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
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
	}))

	config.RouterHooks.Apply(r)

	// Serve the OpenAPI spec
	r.Get("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, swagger)
	})

	// Use validator middleware to check requests against the OpenAPI schema
	_ = api.HandlerWithOptions(impl, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			config.RouterConfig.ErrorHandler.HandleContext(r.Context(), err)
			errorHandlerReply(w, r, err)
		},
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
