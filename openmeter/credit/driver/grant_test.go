package creditdriver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listGrantsRepo struct {
	grant.Repo

	params *grant.ListParams
}

func (r *listGrantsRepo) ListGrants(_ context.Context, params grant.ListParams) (pagination.Result[grant.Grant], error) {
	r.params = &params

	return pagination.Result[grant.Grant]{}, errors.New("stop after capturing the params")
}

func TestListGrantsPageDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params apiv1models.ListGrantsParams
		want   pagination.Page
	}{
		{
			// Neither parameter keeps the unpaged, plain array response.
			name: "no page parameters",
			want: pagination.Page{},
		},
		{
			name:   "page size without page number",
			params: apiv1models.ListGrantsParams{PageSize: lo.ToPtr(10)},
			want:   pagination.Page{PageNumber: 1, PageSize: 10},
		},
		{
			name:   "page number without page size",
			params: apiv1models.ListGrantsParams{Page: lo.ToPtr(2)},
			want:   pagination.Page{PageNumber: 2, PageSize: commonhttp.DefaultPageSize},
		},
		{
			name:   "both page parameters",
			params: apiv1models.ListGrantsParams{Page: lo.ToPtr(3), PageSize: lo.ToPtr(50)},
			want:   pagination.Page{PageNumber: 3, PageSize: 50},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &listGrantsRepo{}
			handler := NewGrantHandler(namespacedriver.StaticNamespaceDecoder("default"), nil, repo, nil)

			handler.ListGrants().
				With(ListGrantsHandlerParams{Params: tc.params}).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/grants", nil))

			require.NotNil(t, repo.params, "the handler must reach the repository")
			require.Equal(t, tc.want, repo.params.Page)
		})
	}
}
