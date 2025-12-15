package oasmiddleware

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// ValidationRouterOpts represents the options to be passed to the validation router for processing the requests.
type ValidationRouterOpts struct {
	// DeleteServers removes the `Servers` property from the parsed OAS spec to be used within test or even at runtime.
	// If you want to use it at runtime please read the doc for `ServerPrefix` first.
	DeleteServers bool
	// ServerPrefix adds a server entry with the desired prefix
	// eg: the service expose domain.com/foo/v0/entity . Your spec path are defined at the /entity level and then
	// /foo/v0 is part of the server entry in the OAS spec. If no prefix is provided, the validation router will either
	// take the whole server entry like domain.com/foo/v0/entity to validate or if `DeleteServers` is to true it will
	// only validate `/entity`
	ServerPrefix string
}

// NewValidationRouter creates a validation router to be injected in the middlewares
// to validate requests or responses. In a case of a bad spec it returns an error
func NewValidationRouter(ctx context.Context, doc *openapi3.T, opts *ValidationRouterOpts) (routers.Router, error) {
	if opts == nil {
		opts = &ValidationRouterOpts{
			DeleteServers: true,
		}
	}

	if opts.DeleteServers {
		doc.Servers = nil

		for key, pathItem := range doc.Paths.Map() {
			pathItem.Servers = nil
			doc.Paths.Set(key, pathItem)
		}
	}

	if opts.ServerPrefix != "" {
		doc.Servers = openapi3.Servers{
			&openapi3.Server{
				URL: opts.ServerPrefix,
			},
		}
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, err
	}

	validationRouter, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, err
	}

	return validationRouter, err
}
