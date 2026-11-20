package request

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
)

func TestParseCursorPage(t *testing.T) {
	// The test cursor is the raw string prefixed with "c:".
	decode := func(cursor string) (*string, error) {
		value, ok := strings.CutPrefix(cursor, "c:")
		if !ok {
			return nil, errors.New("invalid cursor")
		}

		return &value, nil
	}

	for _, tc := range []struct {
		name string
		page *apiv3models.CursorPaginationQueryPage
		want CursorPage[string]
	}{
		{
			name: "no page uses the default size",
			want: CursorPage[string]{Size: 20},
		},
		{
			name: "size",
			page: &apiv3models.CursorPaginationQueryPage{Size: lo.ToPtr(5)},
			want: CursorPage[string]{Size: 5},
		},
		{
			name: "after",
			page: &apiv3models.CursorPaginationQueryPage{After: lo.ToPtr("c:a")},
			want: CursorPage[string]{Size: 20, After: lo.ToPtr("a")},
		},
		{
			name: "before",
			page: &apiv3models.CursorPaginationQueryPage{Before: lo.ToPtr("c:b"), Size: lo.ToPtr(3)},
			want: CursorPage[string]{Size: 3, Before: lo.ToPtr("b")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCursorPage(t.Context(), tc.page, 20, decode)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	for _, tc := range []struct {
		name  string
		page  *apiv3models.CursorPaginationQueryPage
		field string
	}{
		{
			name:  "after and before together",
			page:  &apiv3models.CursorPaginationQueryPage{After: lo.ToPtr("c:a"), Before: lo.ToPtr("c:b")},
			field: "page[after]",
		},
		{
			name:  "invalid after",
			page:  &apiv3models.CursorPaginationQueryPage{After: lo.ToPtr("a")},
			field: "page[after]",
		},
		{
			name:  "invalid before",
			page:  &apiv3models.CursorPaginationQueryPage{Before: lo.ToPtr("b")},
			field: "page[before]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCursorPage(t.Context(), tc.page, 20, decode)

			var apiErr *apierrors.BaseAPIError
			require.True(t, errors.As(err, &apiErr), "expected *apierrors.BaseAPIError, got %T", err)
			require.Equal(t, http.StatusBadRequest, apiErr.Status)
			require.Len(t, apiErr.InvalidParameters, 1)
			require.Equal(t, tc.field, apiErr.InvalidParameters[0].Field)
			require.Equal(t, apierrors.InvalidParamSourceQuery, apiErr.InvalidParameters[0].Source)
		})
	}
}
