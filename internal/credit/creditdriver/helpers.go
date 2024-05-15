package creditdriver

import (
	"context"
	"errors"
	"net/http"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
)

func (h *builder) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.NamespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
