package filter_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestFilterString_SelectJSONB(t *testing.T) {
	tests := []struct {
		name      string
		filter    filter.FilterString
		key       string
		wantEmpty bool
		wantSQL   string
		wantArgs  []any
	}{
		{
			name:      "empty filter",
			filter:    filter.FilterString{},
			key:       "subject.key",
			wantEmpty: true,
		},
		{
			name:     "eq",
			filter:   filter.FilterString{Eq: lo.ToPtr("test")},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) = $2`,
			wantArgs: []any{"subject.key", "test"},
		},
		{
			name:     "ne",
			filter:   filter.FilterString{Ne: lo.ToPtr("test")},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) <> $2`,
			wantArgs: []any{"subject.key", "test"},
		},
		{
			name:     "exists",
			filter:   filter.FilterString{Exists: lo.ToPtr(true)},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) IS NOT NULL`,
			wantArgs: []any{"subject.key"},
		},
		{
			name:     "not exists",
			filter:   filter.FilterString{Exists: lo.ToPtr(false)},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) IS NULL`,
			wantArgs: []any{"subject.key"},
		},
		{
			name:     "in",
			filter:   filter.FilterString{In: &[]string{"a", "b"}},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) IN ($2, $3)`,
			wantArgs: []any{"subject.key", "a", "b"},
		},
		{
			name:     "empty in matches nothing",
			filter:   filter.FilterString{In: &[]string{}},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE FALSE`,
			wantArgs: nil,
		},
		{
			name:     "nin",
			filter:   filter.FilterString{Nin: &[]string{"a", "b"}},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) NOT IN ($2, $3)`,
			wantArgs: []any{"subject.key", "a", "b"},
		},
		{
			name:      "empty nin matches everything",
			filter:    filter.FilterString{Nin: &[]string{}},
			key:       "subject.key",
			wantEmpty: true,
		},
		{
			name:     "contains escapes the pattern",
			filter:   filter.FilterString{Contains: lo.ToPtr("100%")},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) ILIKE $2`,
			wantArgs: []any{"subject.key", `%100\%%`},
		},
		{
			name:     "ncontains",
			filter:   filter.FilterString{Ncontains: lo.ToPtr("test")},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) NOT ILIKE $2`,
			wantArgs: []any{"subject.key", "%test%"},
		},
		{
			name:     "gte",
			filter:   filter.FilterString{Gte: lo.ToPtr("m")},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) >= $2`,
			wantArgs: []any{"subject.key", "m"},
		},
		{
			name: "and of eq and ne",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Ne: lo.ToPtr("a")},
					{Ne: lo.ToPtr("b")},
				},
			},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) <> $2 AND ("test_table"."annotations"->>$3) <> $4`,
			wantArgs: []any{"subject.key", "a", "subject.key", "b"},
		},
		{
			name: "or of eq filters",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("a")},
					{Eq: lo.ToPtr("b")},
				},
			},
			key:      "subject.key",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) = $2 OR ("test_table"."annotations"->>$3) = $4`,
			wantArgs: []any{"subject.key", "a", "subject.key", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate := tt.filter.SelectJSONB("annotations", tt.key)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantSQL, sql)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestFilterULID_SelectJSONB(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterULID
		key      string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "eq through the embedded string filter",
			filter:   filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV")}},
			key:      "subject.id",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) = $2`,
			wantArgs: []any{"subject.id", "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		},
		{
			name: "and combines the children",
			filter: filter.FilterULID{
				And: &[]filter.FilterULID{
					{FilterString: filter.FilterString{Ne: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV")}},
					{FilterString: filter.FilterString{In: &[]string{"01ARZ3NDEKTSV4RRFFQ69G5FAW"}}},
				},
			},
			key:      "subject.id",
			wantSQL:  `SELECT * FROM "test_table" WHERE ("test_table"."annotations"->>$1) <> $2 AND ("test_table"."annotations"->>$3) IN ($4)`,
			wantArgs: []any{"subject.id", "01ARZ3NDEKTSV4RRFFQ69G5FAV", "subject.id", "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate := tt.filter.SelectJSONB("annotations", tt.key)
			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantSQL, sql)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}
