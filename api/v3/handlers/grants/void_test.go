package grants

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

const testGrantID = "01K4WAQ0J99ZZ0MD75HXR112HA"

func serveVoidGrant(t *testing.T, svc fakeService, params api.VoidGrantParams) *httptest.ResponseRecorder {
	t.Helper()

	response := httptest.NewRecorder()
	newTestHandler(svc).VoidGrant().With(VoidGrantParams{
		GrantID: testGrantID,
		Params:  params,
	}).ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/v3/openmeter/grants/"+testGrantID, nil))

	return response
}

func TestVoidGrantHandler(t *testing.T) {
	t.Run("passes the grant and the void time to the service", func(t *testing.T) {
		voidedAt := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

		var received entitlement.VoidGrantInput

		response := serveVoidGrant(t, fakeService{
			voidGrant: func(_ context.Context, input entitlement.VoidGrantInput) error {
				received = input
				return nil
			},
		}, api.VoidGrantParams{VoidedAt: &voidedAt})

		require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
		require.Empty(t, response.Body.String())
		require.Equal(t, entitlement.VoidGrantInput{
			GrantID: models.NamespacedID{Namespace: testNamespace, ID: testGrantID},
			At:      &voidedAt,
		}, received)
	})

	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "maps a missing grant to 404", err: models.NewGenericNotFoundError(errors.New("grant not found")), status: http.StatusNotFound},
		{name: "maps an already voided grant to 400", err: models.NewGenericValidationError(errors.New("grant already voided")), status: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := serveVoidGrant(t, fakeService{
				voidGrant: func(context.Context, entitlement.VoidGrantInput) error { return tc.err },
			}, api.VoidGrantParams{})

			require.Equal(t, tc.status, response.Code, response.Body.String())
		})
	}
}
