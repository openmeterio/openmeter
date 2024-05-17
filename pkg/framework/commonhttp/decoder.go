package commonhttp

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

func JSONRequestBodyDecoder(r *http.Request, out any) error {
	if err := render.DecodeJSON(r.Body, out); err != nil {
		return NewHTTPError(http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
	}
	return nil
}
