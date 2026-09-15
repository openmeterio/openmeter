package billinginvoices

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestUpdateInvoiceRouteEncodesTaxConfigValidationErrorAsBadRequest pins the error-encoding
// contract of the update-invoice route: a tax-config validation error raised inside the edit
// path (a wrapped models.GenericValidationError) must encode as HTTP 400, not fall through
// to the framework's 500 fallback. The route's own encoders (apierrors, billing validation
// issues, billing domain errors) do not handle models.GenericValidationError; the framework
// default commonhttp.GenericErrorEncoder appended by httptransport does.
func TestUpdateInvoiceRouteEncodesTaxConfigValidationErrorAsBadRequest(t *testing.T) {
	// given: a handler wired with the exact error encoder chain of the update-invoice route,
	// whose operation fails with the error shape produced by mapRateCardFromAPI on a
	// tax_config without code, wrapped the way the invoice edit path wraps it.
	innerErr := models.NewGenericValidationError(fmt.Errorf("tax_config.code must be set when tax_config is present"))
	editErr := fmt.Errorf("editing invoice: %w", fmt.Errorf("mapping rate card: %w", innerErr))

	h := httptransport.NewHandler(
		func(_ context.Context, _ *http.Request) (any, error) { return nil, nil },
		func(_ context.Context, _ any) (any, error) { return nil, editErr },
		commonhttp.EmptyResponseEncoder[any](http.StatusOK),
		httptransport.WithOperationName("test"),
		httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		httptransport.WithErrorEncoder(encodeValidationIssue()),
		httptransport.WithErrorEncoder(errorEncoder()),
	)

	// when: the failing operation is served.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/", nil))

	// then: the wrapped validation error encodes as a 400 with the validation message.
	require.Equal(t, http.StatusBadRequest, w.Code, "the default commonhttp error encoder must claim the wrapped validation error before the 500 fallback")
	assert.Contains(t, w.Body.String(), "tax_config.code must be set when tax_config is present")
}
