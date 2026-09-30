package entutils_test

import (
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func TestJSONBFilterString(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterString
		wantSQL  string
		wantArgs []any
		wantNil  bool
		wantErr  bool
	}{
		{name: "empty", filter: filter.FilterString{}, wantNil: true},
		{
			name:     "eq",
			filter:   filter.FilterString{Eq: lo.ToPtr("a")},
			wantSQL:  `SELECT * FROM "t" WHERE ("t"."annotations"->>$1) = $2`,
			wantArgs: []any{"k", "a"},
		},
		{
			name:     "ne",
			filter:   filter.FilterString{Ne: lo.ToPtr("a")},
			wantSQL:  `SELECT * FROM "t" WHERE ("t"."annotations"->>$1) <> $2`,
			wantArgs: []any{"k", "a"},
		},
		{
			name:     "in",
			filter:   filter.FilterString{In: lo.ToPtr([]string{"a", "b"})},
			wantSQL:  `SELECT * FROM "t" WHERE ("t"."annotations"->>$1) IN ($2, $3)`,
			wantArgs: []any{"k", "a", "b"},
		},
		{
			name:    "empty in matches nothing",
			filter:  filter.FilterString{In: lo.ToPtr([]string{})},
			wantSQL: `SELECT * FROM "t" WHERE FALSE`,
		},
		{
			name:     "and",
			filter:   filter.FilterString{And: &[]filter.FilterString{{Eq: lo.ToPtr("a")}, {Ne: lo.ToPtr("b")}}},
			wantSQL:  `SELECT * FROM "t" WHERE ("t"."annotations"->>$1) = $2 AND ("t"."annotations"->>$3) <> $4`,
			wantArgs: []any{"k", "a", "k", "b"},
		},
		{name: "unsupported operator", filter: filter.FilterString{Contains: lo.ToPtr("a")}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pred, err := entutils.JSONBFilterString("annotations", "k", tt.filter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				require.Nil(t, pred)
				return
			}

			s := entsql.Dialect(dialect.Postgres).Select("*").From(entsql.Table("t"))
			pred(s)
			query, args := s.Query()
			require.Equal(t, tt.wantSQL, query)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestJSONBFilterULID(t *testing.T) {
	f := filter.FilterULID{And: &[]filter.FilterULID{
		{FilterString: filter.FilterString{Eq: lo.ToPtr("a")}},
		{FilterString: filter.FilterString{Ne: lo.ToPtr("b")}},
	}}

	pred, err := entutils.JSONBFilterULID("annotations", "k", f)
	require.NoError(t, err)

	s := entsql.Dialect(dialect.Postgres).Select("*").From(entsql.Table("t"))
	pred(s)
	query, args := s.Query()
	require.Equal(t, `SELECT * FROM "t" WHERE ("t"."annotations"->>$1) = $2 AND ("t"."annotations"->>$3) <> $4`, query)
	require.Equal(t, []any{"k", "a", "k", "b"}, args)
}
