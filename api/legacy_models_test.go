package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	cloud "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/api/models"
)

type invoiceListHandler struct {
	api.Unimplemented
	params models.ListInvoicesParams
}

func (h *invoiceListHandler) ListInvoices(w http.ResponseWriter, _ *http.Request, params api.ListInvoicesParams) {
	h.params = params
	w.WriteHeader(http.StatusNoContent)
}

func TestLegacyInvoiceParamsAcrossTransports(t *testing.T) {
	order := cloud.InvoiceOrderByCreatedAt
	params := models.ListInvoicesParams{OrderBy: &order}
	req, err := cloud.NewListInvoicesRequest("https://example.test", &params)
	require.NoError(t, err)

	handler := &invoiceListHandler{}
	resp := httptest.NewRecorder()
	api.Handler(handler).ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)
	require.NotNil(t, handler.params.OrderBy)
	require.Equal(t, api.InvoiceOrderByCreatedAt, *handler.params.OrderBy)
}
