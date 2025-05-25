package commonhttp

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/openmeterio/openmeter/pkg/models"
)

func JSONRequestBodyDecoder(r *http.Request, out any) error {
	if err := render.DecodeJSON(r.Body, out); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("invalid request body: %w", err))
	}
	return nil
}
