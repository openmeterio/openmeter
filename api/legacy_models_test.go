package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	apiv1 "github.com/openmeterio/openmeter/api"
	cloud "github.com/openmeterio/openmeter/api/client/go"
	apiv1models "github.com/openmeterio/openmeter/api/models"
)

type invoiceListHandler struct {
	apiv1.Unimplemented
	params apiv1models.ListInvoicesParams
}

func (h *invoiceListHandler) ListInvoices(w http.ResponseWriter, _ *http.Request, params apiv1.ListInvoicesParams) {
	h.params = params
	w.WriteHeader(http.StatusNoContent)
}

func TestLegacyInvoiceParamsAcrossTransports(t *testing.T) {
	order := cloud.InvoiceOrderByCreatedAt
	params := apiv1models.ListInvoicesParams{OrderBy: &order}
	req, err := cloud.NewListInvoicesRequest("https://example.test", &params)
	require.NoError(t, err)

	handler := &invoiceListHandler{}
	resp := httptest.NewRecorder()
	apiv1.Handler(handler).ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)
	require.NotNil(t, handler.params.OrderBy)
	require.Equal(t, apiv1.InvoiceOrderByCreatedAt, *handler.params.OrderBy)
}
