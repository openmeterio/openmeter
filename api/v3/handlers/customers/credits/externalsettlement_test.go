package customerscredits

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestConvertAPIUpdateCreditGrantExternalSettlementRequest(t *testing.T) {
	req, err := fromAPIUpdateCreditGrantExternalSettlementRequest(
		"ns",
		"cust-1",
		"grant-1",
		api.UpdateCreditGrantExternalSettlementRequest{
			Status: api.BillingCreditPurchasePaymentSettlementStatusAuthorized,
		},
	)

	require.NoError(t, err)
	require.Equal(t, "ns", req.Namespace)
	require.Equal(t, "cust-1", req.CustomerID)
	require.Equal(t, "grant-1", req.ChargeID)
	require.Equal(t, "authorized", string(req.TargetStatus))
}

func TestConvertAPIUpdateCreditGrantExternalSettlementRequestRejectsPending(t *testing.T) {
	_, err := fromAPIUpdateCreditGrantExternalSettlementRequest(
		"ns",
		"cust-1",
		"grant-1",
		api.UpdateCreditGrantExternalSettlementRequest{
			Status: api.BillingCreditPurchasePaymentSettlementStatusPending,
		},
	)

	require.Error(t, err)
	issues, convErr := models.AsValidationIssues(err)
	require.NoError(t, convErr)
	require.Len(t, issues, 1)
	require.ErrorContains(t, err, "unsupported credit grant settlement status")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v3/customers/cust-1/credits/grant-1/external-settlement", nil)
	handled := apierrors.GenericErrorEncoder()(context.Background(), err, rec, req)

	require.True(t, handled)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
