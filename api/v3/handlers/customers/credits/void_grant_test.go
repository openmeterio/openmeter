package customerscredits

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestConvertAPIVoidCreditGrantRequestDefaultsPaymentAdjustment(t *testing.T) {
	req, err := fromAPIVoidCreditGrantRequest(
		"ns",
		"cust-1",
		"grant-1",
		apiv3models.VoidCreditGrantRequest{},
	)

	require.NoError(t, err)
	require.Equal(t, "ns", req.Namespace)
	require.Equal(t, "cust-1", req.CustomerID)
	require.Equal(t, "grant-1", req.ChargeID)
	require.Equal(t, creditgrant.VoidPaymentAdjustmentNone, req.PaymentAdjustment)
}

func TestConvertAPIVoidCreditGrantRequestAcceptsNonePaymentAdjustment(t *testing.T) {
	adjustment := apiv3models.BillingCreditGrantVoidPaymentAdjustmentNone

	req, err := fromAPIVoidCreditGrantRequest(
		"ns",
		"cust-1",
		"grant-1",
		apiv3models.VoidCreditGrantRequest{
			PaymentAdjustment: &adjustment,
		},
	)

	require.NoError(t, err)
	require.Equal(t, creditgrant.VoidPaymentAdjustmentNone, req.PaymentAdjustment)
}

func TestConvertAPIVoidCreditGrantRequestRejectsUnsupportedPaymentAdjustment(t *testing.T) {
	adjustment := apiv3models.BillingCreditGrantVoidPaymentAdjustment("external")

	_, err := fromAPIVoidCreditGrantRequest(
		"ns",
		"cust-1",
		"grant-1",
		apiv3models.VoidCreditGrantRequest{
			PaymentAdjustment: &adjustment,
		},
	)

	require.Error(t, err)
	issues, convErr := models.AsValidationIssues(err)
	require.NoError(t, convErr)
	require.Len(t, issues, 1)
	require.ErrorContains(t, err, "unsupported credit grant void payment adjustment")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v3/customers/cust-1/credits/grants/grant-1/void", nil)
	handled := apierrors.GenericErrorEncoder()(t.Context(), err, rec, req)

	require.True(t, handled)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
