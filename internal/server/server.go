package server

import (
	"fmt"
	"net/http"

	oapimiddleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/server/router"
)

type Server struct {
	chi.Router
}

type ServerLogger struct{}

func (l *ServerLogger) Print(v ...interface{}) {
	slog.Debug(fmt.Sprintf("%v", v[0]), v[1:]...)
}

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

	// override default logger with slog
	// TODO: use https://github.com/go-chi/httplog/tree/master?
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: &ServerLogger{}, NoColor: true})

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Use validator middleware to check requests against the OpenAPI schema
	_ = api.HandlerWithOptions(impl, api.ChiServerOptions{
		BaseRouter: r,
		Middlewares: []api.MiddlewareFunc{
			oapimiddleware.OapiRequestValidator(swagger),
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			_ = render.Render(w, r, api.ErrInternalServerError(err))
		},
	})

	return &Server{
		Router: r,
	}, nil
}
