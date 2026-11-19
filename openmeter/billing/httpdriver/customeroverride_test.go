package httpdriver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listCustomerOverridesService struct {
	billing.Service

	input *billing.ListCustomerBillingProfilesInput
}

func (s *listCustomerOverridesService) ListCustomerBillingProfiles(_ context.Context, input billing.ListCustomerBillingProfilesInput) (billing.ListCustomerBillingProfilesResult, error) {
	s.input = &input

	return billing.ListCustomerBillingProfilesResult{}, errors.New("stop after capturing the input")
}

func TestListCustomerOverridesPageDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params ListCustomerOverridesParams
		want   pagination.Page
	}{
		{
			// Neither parameter keeps listing every override in one response.
			name: "no page parameters",
			want: pagination.Page{},
		},
		{
			name:   "page size without page number",
			params: ListCustomerOverridesParams{PageSize: lo.ToPtr(100)},
			want:   pagination.Page{PageNumber: 1, PageSize: 100},
		},
		{
			name:   "page number without page size",
			params: ListCustomerOverridesParams{Page: lo.ToPtr(2)},
			want:   pagination.Page{PageNumber: 2, PageSize: commonhttp.DefaultPageSize},
		},
		{
			name:   "both page parameters",
			params: ListCustomerOverridesParams{Page: lo.ToPtr(3), PageSize: lo.ToPtr(50)},
			want:   pagination.Page{PageNumber: 3, PageSize: 50},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &listCustomerOverridesService{}
			h := &handler{
				service:          service,
				namespaceDecoder: namespacedriver.StaticNamespaceDecoder("default"),
			}

			h.ListCustomerOverrides().
				With(tc.params).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/billing/customers", nil))

			require.NotNil(t, service.input, "the handler must reach the service")
			require.Equal(t, tc.want, service.input.Page)
		})
	}
}
