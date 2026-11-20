package response

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestNewCursorMeta(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edges CursorPageEdges
		want  string
	}{
		{
			// Next and previous are always present, as null when there is no
			// such page; first and last are omitted for an empty page.
			name: "empty page",
			want: `{"page":{"next":null,"previous":null,"size":10}}`,
		},
		{
			name: "middle page",
			edges: CursorPageEdges{
				First:    lo.ToPtr("f"),
				Last:     lo.ToPtr("l"),
				Next:     lo.ToPtr("n"),
				Previous: lo.ToPtr("p"),
			},
			want: `{"page":{"first":"f","last":"l","next":"n","previous":"p","size":10}}`,
		},
		{
			name: "last page",
			edges: CursorPageEdges{
				First:    lo.ToPtr("f"),
				Last:     lo.ToPtr("l"),
				Previous: lo.ToPtr("p"),
			},
			want: `{"page":{"first":"f","last":"l","next":null,"previous":"p","size":10}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(NewCursorMeta(10, tc.edges))
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(got))
		})
	}
}

type cursorItem struct {
	id string
}

func (i cursorItem) Cursor() pagination.Cursor {
	return pagination.NewCursor(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), i.id)
}

func TestNewCursorPaginationResponse(t *testing.T) {
	cursor := func(id string) string {
		return cursorItem{id: id}.Cursor().Encode()
	}

	t.Run("full page has a next page", func(t *testing.T) {
		resp := NewCursorPaginationResponse([]cursorItem{{"a"}, {"b"}}, 2)

		require.Equal(t, cursor("a"), lo.FromPtr(resp.Meta.Page.First))
		require.Equal(t, cursor("b"), lo.FromPtr(resp.Meta.Page.Last))
		require.Equal(t, cursor("b"), resp.Meta.Page.Next.MustGet())
		require.True(t, resp.Meta.Page.Previous.IsNull())
	})

	t.Run("short page is the last one", func(t *testing.T) {
		resp := NewCursorPaginationResponse([]cursorItem{{"a"}}, 2)

		require.Equal(t, cursor("a"), lo.FromPtr(resp.Meta.Page.Last))
		require.True(t, resp.Meta.Page.Next.IsNull())
	})
}
