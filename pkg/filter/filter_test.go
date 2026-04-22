package filter_test

import (
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/huandu/go-sqlbuilder"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// newSelectBuilder returns an in-memory Ent dialect selector configured for
// Postgres. It is only an SQL builder (analogous to sqlbuilder.SelectBuilder).
func newSelectBuilder() *entsql.Selector {
	return entsql.Dialect(dialect.Postgres).Select("*").From(entsql.Table("test_table"))
}

// assertValidationError asserts on the result of a filter Validate call.
// If wantErr is non-nil, err must be a models.GenericValidationError whose
// underlying cause matches wantErr.
func assertValidationError(t *testing.T, err error, wantErr error) {
	t.Helper()
	if wantErr == nil {
		assert.NoError(t, err)
		return
	}
	if !assert.Error(t, err) {
		return
	}
	assert.True(t, models.IsGenericValidationError(err), "expected a models.GenericValidationError, got %T: %v", err, err)
	assert.True(t, errors.Is(err, wantErr), "expected error to wrap %v, got %v", wantErr, err)
}

func TestEscapeLikePattern(t *testing.T) {
	assert.Equal(t, "", filter.EscapeLikePattern(""))
	assert.Equal(t, "plain", filter.EscapeLikePattern("plain"))
	assert.Equal(t, `100\%\\path\_name`, filter.EscapeLikePattern(`100%\path_name`))
}

func TestContainsPattern(t *testing.T) {
	assert.Equal(t, "%plain%", filter.ContainsPattern("plain"))
	assert.Equal(t, `%a\_b\\c\%d%`, filter.ContainsPattern(`a_b\c%d`))
}

func TestFilterString_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  filter.FilterString
		wantErr error
	}{
		{
			name:   "nil filter",
			filter: filter.FilterString{},
		},
		{
			name: "valid eq filter",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
			},
		},
		{
			name: "valid ne filter",
			filter: filter.FilterString{
				Ne: lo.ToPtr("test"),
			},
		},
		{
			name: "valid exists filter",
			filter: filter.FilterString{
				Exists: lo.ToPtr(true),
			},
		},
		{
			name: "valid exists filter false",
			filter: filter.FilterString{
				Exists: lo.ToPtr(false),
			},
		},
		{
			name: "exists with eq filter",
			filter: filter.FilterString{
				Exists: lo.ToPtr(true),
				Eq:     lo.ToPtr("test"),
			},
			wantErr: filter.ErrFilterMultipleOperators,
		},
		{
			name: "valid in filter",
			filter: filter.FilterString{
				In: &[]string{"test1", "test2"},
			},
		},
		{
			name: "valid nin filter",
			filter: filter.FilterString{
				Nin: &[]string{"test1", "test2"},
			},
		},
		{
			name: "valid like filter",
			filter: filter.FilterString{
				Like: lo.ToPtr("%test%"),
			},
		},
		{
			name: "valid ilike filter",
			filter: filter.FilterString{
				Ilike: lo.ToPtr("%test%"),
			},
		},
		{
			name: "valid gt filter",
			filter: filter.FilterString{
				Gt: lo.ToPtr("test"),
			},
		},
		{
			name: "valid And filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
		},
		{
			name: "valid Or filter",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
		},
		{
			name: "nested And filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						And: &[]filter.FilterString{
							{Eq: lo.ToPtr("test")},
						},
					},
				},
			},
		},
		{
			name: "nested Or filter",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{
						Or: &[]filter.FilterString{
							{Eq: lo.ToPtr("test")},
						},
					},
				},
			},
		},
		{
			name: "multiple filters set",
			filter: filter.FilterString{
				Eq:  lo.ToPtr("test"),
				Ne:  lo.ToPtr("test"),
				Gt:  lo.ToPtr("test"),
				Gte: lo.ToPtr("test"),
				Lt:  lo.ToPtr("test"),
				Lte: lo.ToPtr("test"),
			},
			wantErr: filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.Validate(), tt.wantErr)
		})
	}
}

func TestFilterString_SelectAndSelectWhereExpr(t *testing.T) {
	tests := []struct {
		name         string
		filter       filter.FilterString
		field        string
		wantEmpty    bool
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterString{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name:         "eq filter",
			filter:       filter.FilterString{Eq: lo.ToPtr("test")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{"test"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{"test"},
		},
		{
			name:         "ne filter",
			filter:       filter.FilterString{Ne: lo.ToPtr("test")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field <> ?",
			wantExprArgs: []any{"test"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" <> $1`,
			wantEntArgs:  []any{"test"},
		},
		{
			name:         "exists filter",
			filter:       filter.FilterString{Exists: lo.ToPtr(true)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field IS NOT NULL",
			wantExprArgs: nil,
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" IS NOT NULL`,
			wantEntArgs:  nil,
		},
		{
			name:         "not exists filter",
			filter:       filter.FilterString{Exists: lo.ToPtr(false)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field IS NULL",
			wantExprArgs: nil,
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" IS NULL`,
			wantEntArgs:  nil,
		},
		{
			name:         "in filter",
			filter:       filter.FilterString{In: &[]string{"test1", "test2"}},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field IN (?)",
			wantExprArgs: []any{[]string{"test1", "test2"}},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" IN ($1, $2)`,
			wantEntArgs:  []any{"test1", "test2"},
		},
		{
			name:         "in with single element",
			filter:       filter.FilterString{In: &[]string{"only"}},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field IN (?)",
			wantExprArgs: []any{[]string{"only"}},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" IN ($1)`,
			wantEntArgs:  []any{"only"},
		},
		{
			name:         "nin filter",
			filter:       filter.FilterString{Nin: &[]string{"test1", "test2"}},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field NOT IN (?)",
			wantExprArgs: []any{[]string{"test1", "test2"}},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" NOT IN ($1, $2)`,
			wantEntArgs:  []any{"test1", "test2"},
		},
		{
			name:         "like filter",
			filter:       filter.FilterString{Like: lo.ToPtr("%test%")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field LIKE ?",
			wantExprArgs: []any{"%test%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" LIKE $1`,
			wantEntArgs:  []any{"%test%"},
		},
		{
			name:         "nlike filter",
			filter:       filter.FilterString{Nlike: lo.ToPtr("%test%")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field NOT LIKE ?",
			wantExprArgs: []any{"%test%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" NOT LIKE $1`,
			wantEntArgs:  []any{"%test%"},
		},
		{
			name:         "ilike filter",
			filter:       filter.FilterString{Ilike: lo.ToPtr("%test%")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE LOWER(test_field) LIKE LOWER(?)",
			wantExprArgs: []any{"%test%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" ILIKE $1`,
			wantEntArgs:  []any{"%test%"},
		},
		{
			name:         "nilike filter",
			filter:       filter.FilterString{Nilike: lo.ToPtr("%test%")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE LOWER(test_field) NOT LIKE LOWER(?)",
			wantExprArgs: []any{"%test%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" NOT ILIKE $1`,
			wantEntArgs:  []any{"%test%"},
		},
		{
			name:         "contains filter",
			filter:       filter.FilterString{Contains: lo.ToPtr("needle")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE LOWER(test_field) LIKE LOWER(?)",
			wantExprArgs: []any{"%needle%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" ILIKE $1`,
			wantEntArgs:  []any{"%needle%"},
		},
		{
			name:         "contains filter escapes like metacharacters",
			filter:       filter.FilterString{Contains: lo.ToPtr(`100%_path\name`)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE LOWER(test_field) LIKE LOWER(?)",
			wantExprArgs: []any{`%100\%\_path\\name%`},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" ILIKE $1`,
			wantEntArgs:  []any{`%100\%\_path\\name%`},
		},
		{
			name:         "ncontains filter",
			filter:       filter.FilterString{Ncontains: lo.ToPtr("needle")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE LOWER(test_field) NOT LIKE LOWER(?)",
			wantExprArgs: []any{"%needle%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" NOT ILIKE $1`,
			wantEntArgs:  []any{"%needle%"},
		},
		{
			name:         "ncontains filter escapes like metacharacters",
			filter:       filter.FilterString{Ncontains: lo.ToPtr(`50%`)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE LOWER(test_field) NOT LIKE LOWER(?)",
			wantExprArgs: []any{`%50\%%`},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" NOT ILIKE $1`,
			wantEntArgs:  []any{`%50\%%`},
		},
		{
			name:         "gt filter",
			filter:       filter.FilterString{Gt: lo.ToPtr("test")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field > ?",
			wantExprArgs: []any{"test"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" > $1`,
			wantEntArgs:  []any{"test"},
		},
		{
			name:         "gte filter",
			filter:       filter.FilterString{Gte: lo.ToPtr("test")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field >= ?",
			wantExprArgs: []any{"test"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" >= $1`,
			wantEntArgs:  []any{"test"},
		},
		{
			name:         "lt filter",
			filter:       filter.FilterString{Lt: lo.ToPtr("test")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field < ?",
			wantExprArgs: []any{"test"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" < $1`,
			wantEntArgs:  []any{"test"},
		},
		{
			name:         "lte filter",
			filter:       filter.FilterString{Lte: lo.ToPtr("test")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field <= ?",
			wantExprArgs: []any{"test"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" <= $1`,
			wantEntArgs:  []any{"test"},
		},
		{
			name:         "eq with empty string",
			filter:       filter.FilterString{Eq: lo.ToPtr("")},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{""},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{""},
		},
		{
			name: "and filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field = ? AND test_field = ?)",
			wantExprArgs: []any{"test1", "test2"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 AND "test_table"."test_field" = $2`,
			wantEntArgs:  []any{"test1", "test2"},
		},
		{
			name: "or filter",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field = ? OR test_field = ?)",
			wantExprArgs: []any{"test1", "test2"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 OR "test_table"."test_field" = $2`,
			wantEntArgs:  []any{"test1", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
			} else {
				assert.NotEmpty(t, expr, "SQL expression should not be empty")

				q.Where(expr)
				sql, args := q.Build()

				assert.Equal(t, tt.wantExprSQL, sql, "go-sqlbuilder SQL statement should match expected value")
				assert.Equal(t, tt.wantExprArgs, args, "go-sqlbuilder SQL arguments should match expected values")
			}

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil for empty filter")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantEntSQL, sql, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, args, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterString_SelectAndSelectWhereExpr_NestedOperators(t *testing.T) {
	tests := []struct {
		name         string
		filter       filter.FilterString
		field        string
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name: "deeply nested And filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						And: &[]filter.FilterString{
							{
								And: &[]filter.FilterString{
									{Eq: lo.ToPtr("test1")},
									{Eq: lo.ToPtr("test2")},
								},
							},
						},
					},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (((test_field = ? AND test_field = ?)))",
			wantExprArgs: []any{"test1", "test2"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 AND "test_table"."test_field" = $2`,
			wantEntArgs:  []any{"test1", "test2"},
		},
		{
			name: "mixed nested And/Or filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						Or: &[]filter.FilterString{
							{
								And: &[]filter.FilterString{
									{Eq: lo.ToPtr("test1")},
									{Ne: lo.ToPtr("test2")},
								},
							},
							{Eq: lo.ToPtr("test3")},
						},
					},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (((test_field = ? AND test_field <> ?) OR test_field = ?))",
			wantExprArgs: []any{"test1", "test2", "test3"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE ("test_table"."test_field" = $1 AND "test_table"."test_field" <> $2) OR "test_table"."test_field" = $3`,
			wantEntArgs:  []any{"test1", "test2", "test3"},
		},
		{
			name: "Or of Ands",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{
						And: &[]filter.FilterString{
							{Eq: lo.ToPtr("a")},
							{Ne: lo.ToPtr("b")},
						},
					},
					{
						And: &[]filter.FilterString{
							{Gte: lo.ToPtr("m")},
							{Lte: lo.ToPtr("z")},
						},
					},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE ((test_field = ? AND test_field <> ?) OR (test_field >= ? AND test_field <= ?))",
			wantExprArgs: []any{"a", "b", "m", "z"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE ("test_table"."test_field" = $1 AND "test_table"."test_field" <> $2) OR ("test_table"."test_field" >= $3 AND "test_table"."test_field" <= $4)`,
			wantEntArgs:  []any{"a", "b", "m", "z"},
		},
		{
			name: "Or combining ilike and in",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Ilike: lo.ToPtr("%foo%")},
					{In: &[]string{"bar", "baz"}},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (LOWER(test_field) LIKE LOWER(?) OR test_field IN (?))",
			wantExprArgs: []any{"%foo%", []string{"bar", "baz"}},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" ILIKE $1 OR "test_table"."test_field" IN ($2, $3)`,
			wantEntArgs:  []any{"%foo%", "bar", "baz"},
		},
		{
			name: "And of nested Or filters with contains",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						Or: &[]filter.FilterString{
							{Eq: lo.ToPtr("a")},
							{Eq: lo.ToPtr("b")},
						},
					},
					{
						Or: &[]filter.FilterString{
							{Contains: lo.ToPtr("x")},
							{Contains: lo.ToPtr("y")},
						},
					},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE ((test_field = ? OR test_field = ?) AND (LOWER(test_field) LIKE LOWER(?) OR LOWER(test_field) LIKE LOWER(?)))",
			wantExprArgs: []any{"a", "b", "%x%", "%y%"},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE ("test_table"."test_field" = $1 OR "test_table"."test_field" = $2) AND ("test_table"."test_field" ILIKE $3 OR "test_table"."test_field" ILIKE $4)`,
			wantEntArgs:  []any{"a", "b", "%x%", "%y%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if !assert.NotEmpty(t, expr, "SQL expression should not be empty") {
				return
			}

			q.Where(expr)
			exprSQL, exprArgs := q.Build()

			assert.Equal(t, tt.wantExprSQL, exprSQL, "go-sqlbuilder SQL statement should match expected value")
			assert.Equal(t, tt.wantExprArgs, exprArgs, "go-sqlbuilder SQL arguments should match expected values")

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)
			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			entSQL, entArgs := s.Query()

			assert.Equal(t, tt.wantEntSQL, entSQL, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, entArgs, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterInteger_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  filter.FilterInteger
		wantErr error
	}{
		{
			name:   "nil filter",
			filter: filter.FilterInteger{},
		},
		{
			name: "valid eq filter",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
			},
		},
		{
			name: "valid ne filter",
			filter: filter.FilterInteger{
				Ne: lo.ToPtr(42),
			},
		},
		{
			name: "valid gt filter",
			filter: filter.FilterInteger{
				Gt: lo.ToPtr(42),
			},
		},
		{
			name: "valid And filter",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Gt: lo.ToPtr(10)},
				},
			},
		},
		{
			name: "valid Or filter",
			filter: filter.FilterInteger{
				Or: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Lt: lo.ToPtr(100)},
				},
			},
		},
		{
			name: "nested And filter",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{
						And: &[]filter.FilterInteger{
							{Eq: lo.ToPtr(42)},
						},
					},
				},
			},
		},
		{
			name: "multiple filters set",
			filter: filter.FilterInteger{
				Eq:  lo.ToPtr(42),
				Ne:  lo.ToPtr(42),
				Gt:  lo.ToPtr(42),
				Gte: lo.ToPtr(42),
				Lt:  lo.ToPtr(42),
				Lte: lo.ToPtr(42),
			},
			wantErr: filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.Validate(), tt.wantErr)
		})
	}
}

func TestFilterInteger_SelectAndSelectWhereExpr(t *testing.T) {
	tests := []struct {
		name         string
		filter       filter.FilterInteger
		field        string
		wantEmpty    bool
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterInteger{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name:         "eq filter",
			filter:       filter.FilterInteger{Eq: lo.ToPtr(42)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{42},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{42},
		},
		{
			name:         "ne filter",
			filter:       filter.FilterInteger{Ne: lo.ToPtr(42)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field <> ?",
			wantExprArgs: []any{42},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" <> $1`,
			wantEntArgs:  []any{42},
		},
		{
			name:         "gt filter",
			filter:       filter.FilterInteger{Gt: lo.ToPtr(42)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field > ?",
			wantExprArgs: []any{42},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" > $1`,
			wantEntArgs:  []any{42},
		},
		{
			name:         "gte filter",
			filter:       filter.FilterInteger{Gte: lo.ToPtr(42)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field >= ?",
			wantExprArgs: []any{42},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" >= $1`,
			wantEntArgs:  []any{42},
		},
		{
			name:         "lt filter",
			filter:       filter.FilterInteger{Lt: lo.ToPtr(42)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field < ?",
			wantExprArgs: []any{42},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" < $1`,
			wantEntArgs:  []any{42},
		},
		{
			name:         "lte filter",
			filter:       filter.FilterInteger{Lte: lo.ToPtr(42)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field <= ?",
			wantExprArgs: []any{42},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" <= $1`,
			wantEntArgs:  []any{42},
		},
		{
			name:         "eq filter with zero",
			filter:       filter.FilterInteger{Eq: lo.ToPtr(0)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{0},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{0},
		},
		{
			name:         "eq filter with negative value",
			filter:       filter.FilterInteger{Eq: lo.ToPtr(-7)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{-7},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{-7},
		},
		{
			name: "and filter",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Gt: lo.ToPtr(10)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field = ? AND test_field > ?)",
			wantExprArgs: []any{42, 10},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 AND "test_table"."test_field" > $2`,
			wantEntArgs:  []any{42, 10},
		},
		{
			name: "or filter",
			filter: filter.FilterInteger{
				Or: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Lt: lo.ToPtr(100)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field = ? OR test_field < ?)",
			wantExprArgs: []any{42, 100},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 OR "test_table"."test_field" < $2`,
			wantEntArgs:  []any{42, 100},
		},
		{
			name: "range via and (gte/lte)",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Gte: lo.ToPtr(1)},
					{Lte: lo.ToPtr(100)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field >= ? AND test_field <= ?)",
			wantExprArgs: []any{1, 100},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" >= $1 AND "test_table"."test_field" <= $2`,
			wantEntArgs:  []any{1, 100},
		},
		{
			name: "nested And of Or",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{
						Or: &[]filter.FilterInteger{
							{Eq: lo.ToPtr(1)},
							{Eq: lo.ToPtr(2)},
						},
					},
					{Ne: lo.ToPtr(99)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE ((test_field = ? OR test_field = ?) AND test_field <> ?)",
			wantExprArgs: []any{1, 2, 99},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE ("test_table"."test_field" = $1 OR "test_table"."test_field" = $2) AND "test_table"."test_field" <> $3`,
			wantEntArgs:  []any{1, 2, 99},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
			} else {
				assert.NotEmpty(t, expr, "SQL expression should not be empty")

				q.Where(expr)
				sql, args := q.Build()

				assert.Equal(t, tt.wantExprSQL, sql, "go-sqlbuilder SQL statement should match expected value")
				assert.Equal(t, tt.wantExprArgs, args, "go-sqlbuilder SQL arguments should match expected values")
			}

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil for empty filter")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantEntSQL, sql, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, args, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterFloat_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  filter.FilterFloat
		wantErr error
	}{
		{
			name:   "nil filter",
			filter: filter.FilterFloat{},
		},
		{
			name: "valid eq filter",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
			},
		},
		{
			name: "valid ne filter",
			filter: filter.FilterFloat{
				Ne: lo.ToPtr(42.5),
			},
		},
		{
			name: "valid gt filter",
			filter: filter.FilterFloat{
				Gt: lo.ToPtr(42.5),
			},
		},
		{
			name: "valid And filter",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Gt: lo.ToPtr(10.5)},
				},
			},
		},
		{
			name: "valid Or filter",
			filter: filter.FilterFloat{
				Or: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Lt: lo.ToPtr(100.5)},
				},
			},
		},
		{
			name: "nested And filter",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{
						And: &[]filter.FilterFloat{
							{Eq: lo.ToPtr(42.5)},
						},
					},
				},
			},
		},
		{
			name: "multiple filters set",
			filter: filter.FilterFloat{
				Eq:  lo.ToPtr(42.5),
				Ne:  lo.ToPtr(42.5),
				Gt:  lo.ToPtr(42.5),
				Gte: lo.ToPtr(42.5),
				Lt:  lo.ToPtr(42.5),
				Lte: lo.ToPtr(42.5),
			},
			wantErr: filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.Validate(), tt.wantErr)
		})
	}
}

func TestFilterFloat_SelectAndSelectWhereExpr(t *testing.T) {
	tests := []struct {
		name         string
		filter       filter.FilterFloat
		field        string
		wantEmpty    bool
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterFloat{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name:         "eq filter",
			filter:       filter.FilterFloat{Eq: lo.ToPtr(42.5)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{42.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{42.5},
		},
		{
			name:         "ne filter",
			filter:       filter.FilterFloat{Ne: lo.ToPtr(42.5)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field <> ?",
			wantExprArgs: []any{42.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" <> $1`,
			wantEntArgs:  []any{42.5},
		},
		{
			name:         "gt filter",
			filter:       filter.FilterFloat{Gt: lo.ToPtr(42.5)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field > ?",
			wantExprArgs: []any{42.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" > $1`,
			wantEntArgs:  []any{42.5},
		},
		{
			name:         "gte filter",
			filter:       filter.FilterFloat{Gte: lo.ToPtr(42.5)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field >= ?",
			wantExprArgs: []any{42.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" >= $1`,
			wantEntArgs:  []any{42.5},
		},
		{
			name:         "lt filter",
			filter:       filter.FilterFloat{Lt: lo.ToPtr(42.5)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field < ?",
			wantExprArgs: []any{42.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" < $1`,
			wantEntArgs:  []any{42.5},
		},
		{
			name:         "lte filter",
			filter:       filter.FilterFloat{Lte: lo.ToPtr(42.5)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field <= ?",
			wantExprArgs: []any{42.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" <= $1`,
			wantEntArgs:  []any{42.5},
		},
		{
			name:         "eq filter with zero",
			filter:       filter.FilterFloat{Eq: lo.ToPtr(0.0)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{0.0},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{0.0},
		},
		{
			name:         "eq filter with negative value",
			filter:       filter.FilterFloat{Eq: lo.ToPtr(-3.14)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{-3.14},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1`,
			wantEntArgs:  []any{-3.14},
		},
		{
			name: "and filter",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Gt: lo.ToPtr(10.5)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field = ? AND test_field > ?)",
			wantExprArgs: []any{42.5, 10.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 AND "test_table"."test_field" > $2`,
			wantEntArgs:  []any{42.5, 10.5},
		},
		{
			name: "or filter",
			filter: filter.FilterFloat{
				Or: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Lt: lo.ToPtr(100.5)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field = ? OR test_field < ?)",
			wantExprArgs: []any{42.5, 100.5},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" = $1 OR "test_table"."test_field" < $2`,
			wantEntArgs:  []any{42.5, 100.5},
		},
		{
			name: "range via and (gte/lte)",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Gte: lo.ToPtr(0.0)},
					{Lte: lo.ToPtr(1.0)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE (test_field >= ? AND test_field <= ?)",
			wantExprArgs: []any{0.0, 1.0},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field" >= $1 AND "test_table"."test_field" <= $2`,
			wantEntArgs:  []any{0.0, 1.0},
		},
		{
			name: "nested And of Or",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{
						Or: &[]filter.FilterFloat{
							{Eq: lo.ToPtr(1.5)},
							{Eq: lo.ToPtr(2.5)},
						},
					},
					{Ne: lo.ToPtr(99.0)},
				},
			},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE ((test_field = ? OR test_field = ?) AND test_field <> ?)",
			wantExprArgs: []any{1.5, 2.5, 99.0},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE ("test_table"."test_field" = $1 OR "test_table"."test_field" = $2) AND "test_table"."test_field" <> $3`,
			wantEntArgs:  []any{1.5, 2.5, 99.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
			} else {
				assert.NotEmpty(t, expr, "SQL expression should not be empty")

				q.Where(expr)
				sql, args := q.Build()

				assert.Equal(t, tt.wantExprSQL, sql, "go-sqlbuilder SQL statement should match expected value")
				assert.Equal(t, tt.wantExprArgs, args, "go-sqlbuilder SQL arguments should match expected values")
			}

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil for empty filter")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantEntSQL, sql, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, args, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterBoolean_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  filter.FilterBoolean
		wantErr error
	}{
		{
			name:   "nil filter",
			filter: filter.FilterBoolean{},
		},
		{
			name: "valid eq filter",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.Validate(), tt.wantErr)
		})
	}
}

func TestFilterBoolean_SelectAndSelectWhereExpr(t *testing.T) {
	tests := []struct {
		name         string
		filter       filter.FilterBoolean
		field        string
		wantEmpty    bool
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterBoolean{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name:         "eq filter true",
			filter:       filter.FilterBoolean{Eq: lo.ToPtr(true)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{true},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."test_field"`,
			wantEntArgs:  nil,
		},
		{
			name:         "eq filter false",
			filter:       filter.FilterBoolean{Eq: lo.ToPtr(false)},
			field:        "test_field",
			wantExprSQL:  "SELECT * FROM table WHERE test_field = ?",
			wantExprArgs: []any{false},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE NOT "test_table"."test_field"`,
			wantEntArgs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
			} else {
				assert.NotEmpty(t, expr, "SQL expression should not be empty")

				q.Where(expr)
				sql, args := q.Build()

				assert.Equal(t, tt.wantExprSQL, sql, "go-sqlbuilder SQL statement should match expected value")
				assert.Equal(t, tt.wantExprArgs, args, "go-sqlbuilder SQL arguments should match expected values")
			}

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil for empty filter")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantEntSQL, sql, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, args, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterTime_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  filter.FilterTime
		wantErr error
	}{
		{
			name: "valid single filter",
			filter: filter.FilterTime{
				Gt: lo.ToPtr(time.Now()),
			},
		},
		{
			name: "valid AND filter",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gt: lo.ToPtr(time.Now())},
					{Lt: lo.ToPtr(time.Now().Add(24 * time.Hour))},
				},
			},
		},
		{
			name: "valid OR filter",
			filter: filter.FilterTime{
				Or: &[]filter.FilterTime{
					{Gt: lo.ToPtr(time.Now())},
					{Lt: lo.ToPtr(time.Now().Add(24 * time.Hour))},
				},
			},
		},
		{
			name: "invalid multiple filters",
			filter: filter.FilterTime{
				Gt:  lo.ToPtr(time.Now()),
				Gte: lo.ToPtr(time.Now()),
			},
			wantErr: filter.ErrFilterMultipleOperators,
		},
		{
			name: "invalid nested AND filter",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{
						Gt:  lo.ToPtr(time.Now()),
						Gte: lo.ToPtr(time.Now()),
					},
				},
			},
			wantErr: filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.Validate(), tt.wantErr)
		})
	}
}

func TestFilterTime_SelectAndSelectWhereExpr(t *testing.T) {
	now := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	later := now.Add(24 * time.Hour)

	tests := []struct {
		name         string
		filter       filter.FilterTime
		field        string
		wantEmpty    bool
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterTime{},
			field:     "created_at",
			wantEmpty: true,
		},
		{
			name:         "eq filter",
			filter:       filter.FilterTime{Eq: lo.ToPtr(now)},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at = ?",
			wantExprArgs: []any{now},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" = $1`,
			wantEntArgs:  []any{now},
		},
		{
			name:         "gt filter",
			filter:       filter.FilterTime{Gt: lo.ToPtr(now)},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at > ?",
			wantExprArgs: []any{now},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" > $1`,
			wantEntArgs:  []any{now},
		},
		{
			name:         "gte filter",
			filter:       filter.FilterTime{Gte: lo.ToPtr(now)},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at >= ?",
			wantExprArgs: []any{now},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" >= $1`,
			wantEntArgs:  []any{now},
		},
		{
			name:         "lt filter",
			filter:       filter.FilterTime{Lt: lo.ToPtr(now)},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at < ?",
			wantExprArgs: []any{now},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" < $1`,
			wantEntArgs:  []any{now},
		},
		{
			name:         "lte filter",
			filter:       filter.FilterTime{Lte: lo.ToPtr(now)},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at <= ?",
			wantExprArgs: []any{now},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" <= $1`,
			wantEntArgs:  []any{now},
		},
		{
			name: "and filter",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gt: lo.ToPtr(now)},
					{Lt: lo.ToPtr(later)},
				},
			},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE (created_at > ? AND created_at < ?)",
			wantExprArgs: []any{now, later},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" > $1 AND "test_table"."created_at" < $2`,
			wantEntArgs:  []any{now, later},
		},
		{
			name: "or filter",
			filter: filter.FilterTime{
				Or: &[]filter.FilterTime{
					{Lt: lo.ToPtr(now)},
					{Gt: lo.ToPtr(later)},
				},
			},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE (created_at < ? OR created_at > ?)",
			wantExprArgs: []any{now, later},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" < $1 OR "test_table"."created_at" > $2`,
			wantEntArgs:  []any{now, later},
		},
		{
			name: "range via and (gte/lt)",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gte: lo.ToPtr(now)},
					{Lt: lo.ToPtr(later)},
				},
			},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE (created_at >= ? AND created_at < ?)",
			wantExprArgs: []any{now, later},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" >= $1 AND "test_table"."created_at" < $2`,
			wantEntArgs:  []any{now, later},
		},
		{
			name: "nested And of Or",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{
						Or: &[]filter.FilterTime{
							{Lt: lo.ToPtr(now)},
							{Gt: lo.ToPtr(later.Add(24 * time.Hour))},
						},
					},
					{Gte: lo.ToPtr(now.Add(-24 * time.Hour))},
				},
			},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE ((created_at < ? OR created_at > ?) AND created_at >= ?)",
			wantExprArgs: []any{now, later.Add(24 * time.Hour), now.Add(-24 * time.Hour)},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE ("test_table"."created_at" < $1 OR "test_table"."created_at" > $2) AND "test_table"."created_at" >= $3`,
			wantEntArgs:  []any{now, later.Add(24 * time.Hour), now.Add(-24 * time.Hour)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
			} else {
				assert.NotEmpty(t, expr, "SQL expression should not be empty")

				q.Where(expr)
				sql, args := q.Build()

				assert.Equal(t, tt.wantExprSQL, sql, "go-sqlbuilder SQL statement should match expected value")
				assert.Equal(t, tt.wantExprArgs, args, "go-sqlbuilder SQL arguments should match expected values")
			}

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil for empty filter")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantEntSQL, sql, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, args, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterTimeUnix_SelectAndSelectWhereExpr(t *testing.T) {
	now := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		name         string
		filter       filter.FilterTimeUnix
		field        string
		wantEmpty    bool
		wantExprSQL  string
		wantExprArgs []any
		wantEntSQL   string
		wantEntArgs  []any
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterTimeUnix{},
			field:     "created_at",
			wantEmpty: true,
		},
		{
			name:         "eq filter uses unix seconds",
			filter:       filter.FilterTimeUnix{FilterTime: filter.FilterTime{Eq: lo.ToPtr(now)}},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at = ?",
			wantExprArgs: []any{now.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" = $1`,
			wantEntArgs:  []any{now.Unix()},
		},
		{
			name:         "gt filter uses unix seconds",
			filter:       filter.FilterTimeUnix{FilterTime: filter.FilterTime{Gt: lo.ToPtr(now)}},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at > ?",
			wantExprArgs: []any{now.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" > $1`,
			wantEntArgs:  []any{now.Unix()},
		},
		{
			name:         "gte filter uses unix seconds",
			filter:       filter.FilterTimeUnix{FilterTime: filter.FilterTime{Gte: lo.ToPtr(now)}},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at >= ?",
			wantExprArgs: []any{now.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" >= $1`,
			wantEntArgs:  []any{now.Unix()},
		},
		{
			name:         "lt filter uses unix seconds",
			filter:       filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lt: lo.ToPtr(now)}},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at < ?",
			wantExprArgs: []any{now.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" < $1`,
			wantEntArgs:  []any{now.Unix()},
		},
		{
			name:         "lte filter uses unix seconds",
			filter:       filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lte: lo.ToPtr(now)}},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE created_at <= ?",
			wantExprArgs: []any{now.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" <= $1`,
			wantEntArgs:  []any{now.Unix()},
		},
		{
			name: "and filter uses unix seconds",
			filter: filter.FilterTimeUnix{
				FilterTime: filter.FilterTime{
					And: &[]filter.FilterTime{
						{Gte: lo.ToPtr(now)},
						{Lt: lo.ToPtr(later)},
					},
				},
			},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE (created_at >= ? AND created_at < ?)",
			wantExprArgs: []any{now.Unix(), later.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" >= $1 AND "test_table"."created_at" < $2`,
			wantEntArgs:  []any{now.Unix(), later.Unix()},
		},
		{
			name: "or filter uses unix seconds",
			filter: filter.FilterTimeUnix{
				FilterTime: filter.FilterTime{
					Or: &[]filter.FilterTime{
						{Lt: lo.ToPtr(now)},
						{Gt: lo.ToPtr(later)},
					},
				},
			},
			field:        "created_at",
			wantExprSQL:  "SELECT * FROM table WHERE (created_at < ? OR created_at > ?)",
			wantExprArgs: []any{now.Unix(), later.Unix()},
			wantEntSQL:   `SELECT * FROM "test_table" WHERE "test_table"."created_at" < $1 OR "test_table"."created_at" > $2`,
			wantEntArgs:  []any{now.Unix(), later.Unix()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SelectWhereExpr (go-sqlbuilder) branch.
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
			} else {
				assert.NotEmpty(t, expr, "SQL expression should not be empty")

				q.Where(expr)
				sql, args := q.Build()

				assert.Equal(t, tt.wantExprSQL, sql, "go-sqlbuilder SQL statement should match expected value")
				assert.Equal(t, tt.wantExprArgs, args, "go-sqlbuilder SQL arguments should match expected values")
			}

			// Select (Ent) branch.
			predicate := tt.filter.Select(tt.field)

			if tt.wantEmpty {
				assert.Nil(t, predicate, "predicate should be nil for empty filter")
				return
			}

			if !assert.NotNil(t, predicate, "predicate should not be nil") {
				return
			}

			s := newSelectBuilder()
			predicate(s)
			sql, args := s.Query()

			assert.Equal(t, tt.wantEntSQL, sql, "Ent SQL statement should match expected value")
			assert.Equal(t, tt.wantEntArgs, args, "Ent SQL arguments should match expected values")
		})
	}
}

func TestFilterString_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterString
		maxDepth int
		wantErr  error
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterString{},
			maxDepth: 3,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
			},
			maxDepth: 3,
		},
		{
			name: "one level nested AND filter within depth limit",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			maxDepth: 3,
		},
		{
			name: "one level nested OR filter within depth limit",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			maxDepth: 3,
		},
		{
			name: "two level nested AND filter within depth limit",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						And: &[]filter.FilterString{
							{Eq: lo.ToPtr("test1")},
							{Eq: lo.ToPtr("test2")},
						},
					},
				},
			},
			maxDepth: 3,
		},
		{
			name: "deep nested filter exceeding depth limit",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						And: &[]filter.FilterString{
							{
								And: &[]filter.FilterString{
									{
										And: &[]filter.FilterString{
											{Eq: lo.ToPtr("test1")},
										},
									},
								},
							},
						},
					},
				},
			},
			maxDepth: 2,
			wantErr:  filter.ErrFilterComplexityExceeded,
		},
		{
			name: "mixed nested AND/OR filter within depth limit",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						Or: &[]filter.FilterString{
							{Eq: lo.ToPtr("test1")},
							{Eq: lo.ToPtr("test2")},
						},
					},
				},
			},
			maxDepth: 3,
		},
		{
			name: "mixed nested AND/OR filter exceeding depth limit",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{
						Or: &[]filter.FilterString{
							{
								And: &[]filter.FilterString{
									{Eq: lo.ToPtr("test1")},
									{Eq: lo.ToPtr("test2")},
								},
							},
						},
					},
				},
			},
			maxDepth: 2,
			wantErr:  filter.ErrFilterComplexityExceeded,
		},
		{
			name: "filter with validation error",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
				Ne: lo.ToPtr("test"),
			},
			maxDepth: 3,
			wantErr:  filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.ValidateWithComplexity(tt.maxDepth), tt.wantErr)
		})
	}
}

func TestFilterInteger_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterInteger
		maxDepth int
		wantErr  error
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterInteger{},
			maxDepth: 3,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
			},
			maxDepth: 3,
		},
		{
			name: "one level nested AND filter within depth limit",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Gt: lo.ToPtr(10)},
				},
			},
			maxDepth: 3,
		},
		{
			name: "one level nested OR filter within depth limit",
			filter: filter.FilterInteger{
				Or: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Lt: lo.ToPtr(100)},
				},
			},
			maxDepth: 3,
		},
		{
			name: "two level nested AND filter within depth limit",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{
						And: &[]filter.FilterInteger{
							{Eq: lo.ToPtr(42)},
							{Gt: lo.ToPtr(10)},
						},
					},
				},
			},
			maxDepth: 3,
		},
		{
			name: "deep nested filter exceeding depth limit",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{
						And: &[]filter.FilterInteger{
							{
								And: &[]filter.FilterInteger{
									{
										And: &[]filter.FilterInteger{
											{Eq: lo.ToPtr(42)},
										},
									},
								},
							},
						},
					},
				},
			},
			maxDepth: 2,
			wantErr:  filter.ErrFilterComplexityExceeded,
		},
		{
			name: "mixed nested AND/OR filter within depth limit",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{
						Or: &[]filter.FilterInteger{
							{Eq: lo.ToPtr(42)},
							{Gt: lo.ToPtr(10)},
						},
					},
				},
			},
			maxDepth: 3,
		},
		{
			name: "filter with validation error",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
				Ne: lo.ToPtr(42),
			},
			maxDepth: 3,
			wantErr:  filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.ValidateWithComplexity(tt.maxDepth), tt.wantErr)
		})
	}
}

func TestFilterFloat_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterFloat
		maxDepth int
		wantErr  error
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterFloat{},
			maxDepth: 3,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
			},
			maxDepth: 3,
		},
		{
			name: "one level nested AND filter within depth limit",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Gt: lo.ToPtr(10.5)},
				},
			},
			maxDepth: 3,
		},
		{
			name: "two level nested AND filter within depth limit",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{
						And: &[]filter.FilterFloat{
							{Eq: lo.ToPtr(42.5)},
							{Gt: lo.ToPtr(10.5)},
						},
					},
				},
			},
			maxDepth: 3,
		},
		{
			name: "deep nested filter exceeding depth limit",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{
						And: &[]filter.FilterFloat{
							{
								And: &[]filter.FilterFloat{
									{
										And: &[]filter.FilterFloat{
											{Eq: lo.ToPtr(42.5)},
										},
									},
								},
							},
						},
					},
				},
			},
			maxDepth: 2,
			wantErr:  filter.ErrFilterComplexityExceeded,
		},
		{
			name: "filter with validation error",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
				Ne: lo.ToPtr(42.5),
			},
			maxDepth: 3,
			wantErr:  filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.ValidateWithComplexity(tt.maxDepth), tt.wantErr)
		})
	}
}

func TestFilterBoolean_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterBoolean
		maxDepth int
		wantErr  error
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterBoolean{},
			maxDepth: 3,
		},
		{
			name: "simple filter",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(true),
			},
			maxDepth: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.ValidateWithComplexity(tt.maxDepth), tt.wantErr)
		})
	}
}

func TestFilterTime_ValidateWithComplexity(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		filter   filter.FilterTime
		maxDepth int
		wantErr  error
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterTime{},
			maxDepth: 3,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterTime{
				Gt: lo.ToPtr(now),
			},
			maxDepth: 3,
		},
		{
			name: "one level nested AND filter within depth limit",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gt: lo.ToPtr(now)},
					{Lt: lo.ToPtr(now.Add(24 * time.Hour))},
				},
			},
			maxDepth: 3,
		},
		{
			name: "two level nested AND filter within depth limit",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{
						And: &[]filter.FilterTime{
							{Gt: lo.ToPtr(now)},
							{Lt: lo.ToPtr(now.Add(24 * time.Hour))},
						},
					},
				},
			},
			maxDepth: 3,
		},
		{
			name: "deep nested filter exceeding depth limit",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{
						And: &[]filter.FilterTime{
							{
								And: &[]filter.FilterTime{
									{
										And: &[]filter.FilterTime{
											{Gt: lo.ToPtr(now)},
										},
									},
								},
							},
						},
					},
				},
			},
			maxDepth: 2,
			wantErr:  filter.ErrFilterComplexityExceeded,
		},
		{
			name: "filter with validation error",
			filter: filter.FilterTime{
				Gt:  lo.ToPtr(now),
				Gte: lo.ToPtr(now),
			},
			maxDepth: 3,
			wantErr:  filter.ErrFilterMultipleOperators,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationError(t, tt.filter.ValidateWithComplexity(tt.maxDepth), tt.wantErr)
		})
	}
}

func TestFilterString_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter filter.FilterString
		want   bool
	}{
		{
			name:   "empty filter",
			filter: filter.FilterString{},
			want:   true,
		},
		{
			name: "eq filter",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
			},
			want: false,
		},
		{
			name: "ne filter",
			filter: filter.FilterString{
				Ne: lo.ToPtr("test"),
			},
			want: false,
		},
		{
			name: "exists filter",
			filter: filter.FilterString{
				Exists: lo.ToPtr(true),
			},
			want: false,
		},
		{
			name: "in filter",
			filter: filter.FilterString{
				In: &[]string{"test1", "test2"},
			},
			want: false,
		},
		{
			name: "nin filter",
			filter: filter.FilterString{
				Nin: &[]string{"test1", "test2"},
			},
			want: false,
		},
		{
			name: "like filter",
			filter: filter.FilterString{
				Like: lo.ToPtr("%test%"),
			},
			want: false,
		},
		{
			name: "nlike filter",
			filter: filter.FilterString{
				Nlike: lo.ToPtr("%test%"),
			},
			want: false,
		},
		{
			name: "ilike filter",
			filter: filter.FilterString{
				Ilike: lo.ToPtr("%test%"),
			},
			want: false,
		},
		{
			name: "nilike filter",
			filter: filter.FilterString{
				Nilike: lo.ToPtr("%test%"),
			},
			want: false,
		},
		{
			name: "gt filter",
			filter: filter.FilterString{
				Gt: lo.ToPtr("test"),
			},
			want: false,
		},
		{
			name: "gte filter",
			filter: filter.FilterString{
				Gte: lo.ToPtr("test"),
			},
			want: false,
		},
		{
			name: "lt filter",
			filter: filter.FilterString{
				Lt: lo.ToPtr("test"),
			},
			want: false,
		},
		{
			name: "lte filter",
			filter: filter.FilterString{
				Lte: lo.ToPtr("test"),
			},
			want: false,
		},
		{
			name: "and filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Eq: lo.ToPtr("test")},
				},
			},
			want: false,
		},
		{
			name: "or filter",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("test")},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterInteger_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter filter.FilterInteger
		want   bool
	}{
		{
			name:   "empty filter",
			filter: filter.FilterInteger{},
			want:   true,
		},
		{
			name: "eq filter",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
			},
			want: false,
		},
		{
			name: "ne filter",
			filter: filter.FilterInteger{
				Ne: lo.ToPtr(42),
			},
			want: false,
		},
		{
			name: "gt filter",
			filter: filter.FilterInteger{
				Gt: lo.ToPtr(42),
			},
			want: false,
		},
		{
			name: "gte filter",
			filter: filter.FilterInteger{
				Gte: lo.ToPtr(42),
			},
			want: false,
		},
		{
			name: "lt filter",
			filter: filter.FilterInteger{
				Lt: lo.ToPtr(42),
			},
			want: false,
		},
		{
			name: "lte filter",
			filter: filter.FilterInteger{
				Lte: lo.ToPtr(42),
			},
			want: false,
		},
		{
			name: "and filter",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
				},
			},
			want: false,
		},
		{
			name: "or filter",
			filter: filter.FilterInteger{
				Or: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterFloat_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter filter.FilterFloat
		want   bool
	}{
		{
			name:   "empty filter",
			filter: filter.FilterFloat{},
			want:   true,
		},
		{
			name: "eq filter",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
			},
			want: false,
		},
		{
			name: "ne filter",
			filter: filter.FilterFloat{
				Ne: lo.ToPtr(42.5),
			},
			want: false,
		},
		{
			name: "gt filter",
			filter: filter.FilterFloat{
				Gt: lo.ToPtr(42.5),
			},
			want: false,
		},
		{
			name: "gte filter",
			filter: filter.FilterFloat{
				Gte: lo.ToPtr(42.5),
			},
			want: false,
		},
		{
			name: "lt filter",
			filter: filter.FilterFloat{
				Lt: lo.ToPtr(42.5),
			},
			want: false,
		},
		{
			name: "lte filter",
			filter: filter.FilterFloat{
				Lte: lo.ToPtr(42.5),
			},
			want: false,
		},
		{
			name: "and filter",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
				},
			},
			want: false,
		},
		{
			name: "or filter",
			filter: filter.FilterFloat{
				Or: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterBoolean_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter filter.FilterBoolean
		want   bool
	}{
		{
			name:   "empty filter",
			filter: filter.FilterBoolean{},
			want:   true,
		},
		{
			name: "eq filter true",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(true),
			},
			want: false,
		},
		{
			name: "eq filter false",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(false),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterTime_IsEmpty(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		filter filter.FilterTime
		want   bool
	}{
		{
			name:   "empty filter",
			filter: filter.FilterTime{},
			want:   true,
		},
		{
			name: "gt filter",
			filter: filter.FilterTime{
				Gt: lo.ToPtr(now),
			},
			want: false,
		},
		{
			name: "gte filter",
			filter: filter.FilterTime{
				Gte: lo.ToPtr(now),
			},
			want: false,
		},
		{
			name: "lt filter",
			filter: filter.FilterTime{
				Lt: lo.ToPtr(now),
			},
			want: false,
		},
		{
			name: "lte filter",
			filter: filter.FilterTime{
				Lte: lo.ToPtr(now),
			},
			want: false,
		},
		{
			name: "and filter",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gt: lo.ToPtr(now)},
				},
			},
			want: false,
		},
		{
			name: "or filter",
			filter: filter.FilterTime{
				Or: &[]filter.FilterTime{
					{Gt: lo.ToPtr(now)},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterULID_Validate(t *testing.T) {
	tests := []struct {
		name   string
		filter filter.FilterULID
		valid  bool
	}{
		{
			name: "valid_single",
			filter: filter.FilterULID{
				FilterString: filter.FilterString{
					Eq: lo.ToPtr(ulid.Make().String()),
				},
			},
			valid: true,
		},
		{
			name: "valid_multiple",
			filter: filter.FilterULID{
				FilterString: filter.FilterString{
					In: lo.ToPtr([]string{ulid.Make().String(), ulid.Make().String(), ulid.Make().String()}),
				},
			},
			valid: true,
		},
		{
			name: "valid_and",
			filter: filter.FilterULID{
				And: lo.ToPtr([]filter.FilterULID{
					{
						FilterString: filter.FilterString{
							Eq: lo.ToPtr(ulid.Make().String()),
						},
					},
					{
						FilterString: filter.FilterString{
							Ne: lo.ToPtr(ulid.Make().String()),
						},
					},
				}),
			},
			valid: true,
		},
		{
			name: "valid_or",
			filter: filter.FilterULID{
				Or: lo.ToPtr([]filter.FilterULID{
					{
						FilterString: filter.FilterString{
							Eq: lo.ToPtr(ulid.Make().String()),
						},
					},
					{
						FilterString: filter.FilterString{
							Ne: lo.ToPtr(ulid.Make().String()),
						},
					},
				}),
			},
			valid: true,
		},
		{
			name: "invalid_single",
			filter: filter.FilterULID{
				FilterString: filter.FilterString{
					Ne: lo.ToPtr("test"),
				},
			},
			valid: false,
		},
		{
			name: "invalid_multiple",
			filter: filter.FilterULID{
				FilterString: filter.FilterString{
					Nin: lo.ToPtr([]string{"test", "test2", "test3"}),
				},
			},
			valid: false,
		},
		{
			name: "invalid_multiple_partial",
			filter: filter.FilterULID{
				FilterString: filter.FilterString{
					In: lo.ToPtr([]string{ulid.Make().String(), "test", ulid.Make().String()}),
				},
			},
			valid: false,
		},
		{
			name: "invalid_and",
			filter: filter.FilterULID{
				And: lo.ToPtr([]filter.FilterULID{
					{
						FilterString: filter.FilterString{
							Eq: lo.ToPtr(ulid.Make().String()),
						},
					},
					{
						FilterString: filter.FilterString{
							Ne: lo.ToPtr("test"),
						},
					},
				}),
			},
			valid: false,
		},
		{
			name: "invalid_or",
			filter: filter.FilterULID{
				Or: lo.ToPtr([]filter.FilterULID{
					{
						FilterString: filter.FilterString{
							Eq: lo.ToPtr(ulid.Make().String()),
						},
					},
					{
						FilterString: filter.FilterString{
							Ne: lo.ToPtr("test"),
						},
					},
				}),
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate()
			if tt.valid {
				assert.NoError(t, got)
			} else {
				assert.ErrorIs(t, got, filter.ErrFilterFormatMismatch)
			}
		})
	}
}
