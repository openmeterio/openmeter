package filter_test

import (
	"testing"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestFilterString_Validate(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterString
		wantErr    bool
		errMessage string
	}{
		{
			name:    "nil filter",
			filter:  filter.FilterString{},
			wantErr: false,
		},
		{
			name: "valid eq filter",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
			},
			wantErr: false,
		},
		{
			name: "valid ne filter",
			filter: filter.FilterString{
				Ne: lo.ToPtr("test"),
			},
			wantErr: false,
		},
		{
			name: "valid in filter",
			filter: filter.FilterString{
				In: &[]string{"test1", "test2"},
			},
			wantErr: false,
		},
		{
			name: "valid nin filter",
			filter: filter.FilterString{
				Nin: &[]string{"test1", "test2"},
			},
			wantErr: false,
		},
		{
			name: "valid like filter",
			filter: filter.FilterString{
				Like: lo.ToPtr("%test%"),
			},
			wantErr: false,
		},
		{
			name: "valid ilike filter",
			filter: filter.FilterString{
				Ilike: lo.ToPtr("%test%"),
			},
			wantErr: false,
		},
		{
			name: "valid gt filter",
			filter: filter.FilterString{
				Gt: lo.ToPtr("test"),
			},
			wantErr: false,
		},
		{
			name: "valid And filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			wantErr: false,
		},
		{
			name: "valid Or filter",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.EqualError(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterString_SelectWhereExpr(t *testing.T) {
	tests := []struct {
		name      string
		filter    filter.FilterString
		field     string
		wantEmpty bool
		wantSQL   string
		wantArgs  []interface{}
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterString{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name: "eq filter",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field = ?",
			wantArgs:  []interface{}{"test"},
		},
		{
			name: "ne filter",
			filter: filter.FilterString{
				Ne: lo.ToPtr("test"),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field <> ?",
			wantArgs:  []interface{}{"test"},
		},
		{
			name: "in filter",
			filter: filter.FilterString{
				In: &[]string{"test1", "test2"},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field IN (?)",
			wantArgs:  []interface{}{[]string{"test1", "test2"}},
		},
		{
			name: "nin filter",
			filter: filter.FilterString{
				Nin: &[]string{"test1", "test2"},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field NOT IN (?)",
			wantArgs:  []interface{}{[]string{"test1", "test2"}},
		},
		{
			name: "like filter",
			filter: filter.FilterString{
				Like: lo.ToPtr("%test%"),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field LIKE ?",
			wantArgs:  []interface{}{"%test%"},
		},
		{
			name: "ilike filter",
			filter: filter.FilterString{
				Ilike: lo.ToPtr("%test%"),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE LOWER(test_field) LIKE LOWER(?)",
			wantArgs:  []interface{}{"%test%"},
		},
		{
			name: "gt filter",
			filter: filter.FilterString{
				Gt: lo.ToPtr("test"),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field > ?",
			wantArgs:  []interface{}{"test"},
		},
		{
			name: "and filter",
			filter: filter.FilterString{
				And: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (test_field = ? AND test_field = ?)",
			wantArgs:  []interface{}{"test1", "test2"},
		},
		{
			name: "or filter",
			filter: filter.FilterString{
				Or: &[]filter.FilterString{
					{Eq: lo.ToPtr("test1")},
					{Eq: lo.ToPtr("test2")},
				},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (test_field = ? OR test_field = ?)",
			wantArgs:  []interface{}{"test1", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
				return
			}

			assert.NotEmpty(t, expr, "SQL expression should not be empty")

			q.Where(expr)
			sql, args := q.Build()

			assert.Equal(t, tt.wantSQL, sql, "SQL statement should match expected value")
			assert.Equal(t, tt.wantArgs, args, "SQL arguments should match expected values")
		})
	}
}

func TestFilterString_SelectWhereExpr_NestedOperators(t *testing.T) {
	tests := []struct {
		name      string
		filter    filter.FilterString
		field     string
		wantEmpty bool
		wantSQL   string
		wantArgs  []interface{}
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
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (((test_field = ? AND test_field = ?)))",
			wantArgs:  []interface{}{"test1", "test2"},
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
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (((test_field = ? AND test_field <> ?) OR test_field = ?))",
			wantArgs:  []interface{}{"test1", "test2", "test3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
				return
			}

			assert.NotEmpty(t, expr, "SQL expression should not be empty")

			q.Where(expr)
			sql, args := q.Build()

			assert.Equal(t, tt.wantSQL, sql, "SQL statement should match expected value")
			assert.Equal(t, tt.wantArgs, args, "SQL arguments should match expected values")
		})
	}
}

func TestFilterInteger_Validate(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterInteger
		wantErr    bool
		errMessage string
	}{
		{
			name:    "nil filter",
			filter:  filter.FilterInteger{},
			wantErr: false,
		},
		{
			name: "valid eq filter",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
			},
			wantErr: false,
		},
		{
			name: "valid ne filter",
			filter: filter.FilterInteger{
				Ne: lo.ToPtr(42),
			},
			wantErr: false,
		},
		{
			name: "valid gt filter",
			filter: filter.FilterInteger{
				Gt: lo.ToPtr(42),
			},
			wantErr: false,
		},
		{
			name: "valid And filter",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Gt: lo.ToPtr(10)},
				},
			},
			wantErr: false,
		},
		{
			name: "valid Or filter",
			filter: filter.FilterInteger{
				Or: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Lt: lo.ToPtr(100)},
				},
			},
			wantErr: false,
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
			wantErr: false,
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
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.EqualError(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterInteger_SelectWhereExpr(t *testing.T) {
	tests := []struct {
		name      string
		filter    filter.FilterInteger
		field     string
		wantEmpty bool
		wantSQL   string
		wantArgs  []interface{}
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterInteger{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name: "eq filter",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field = ?",
			wantArgs:  []interface{}{42},
		},
		{
			name: "ne filter",
			filter: filter.FilterInteger{
				Ne: lo.ToPtr(42),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field <> ?",
			wantArgs:  []interface{}{42},
		},
		{
			name: "gt filter",
			filter: filter.FilterInteger{
				Gt: lo.ToPtr(42),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field > ?",
			wantArgs:  []interface{}{42},
		},
		{
			name: "and filter",
			filter: filter.FilterInteger{
				And: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Gt: lo.ToPtr(10)},
				},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (test_field = ? AND test_field > ?)",
			wantArgs:  []interface{}{42, 10},
		},
		{
			name: "or filter",
			filter: filter.FilterInteger{
				Or: &[]filter.FilterInteger{
					{Eq: lo.ToPtr(42)},
					{Lt: lo.ToPtr(100)},
				},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (test_field = ? OR test_field < ?)",
			wantArgs:  []interface{}{42, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
				return
			}

			assert.NotEmpty(t, expr, "SQL expression should not be empty")

			q.Where(expr)
			sql, args := q.Build()

			assert.Equal(t, tt.wantSQL, sql, "SQL statement should match expected value")
			assert.Equal(t, tt.wantArgs, args, "SQL arguments should match expected values")
		})
	}
}

func TestFilterFloat_Validate(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterFloat
		wantErr    bool
		errMessage string
	}{
		{
			name:    "nil filter",
			filter:  filter.FilterFloat{},
			wantErr: false,
		},
		{
			name: "valid eq filter",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
			},
			wantErr: false,
		},
		{
			name: "valid ne filter",
			filter: filter.FilterFloat{
				Ne: lo.ToPtr(42.5),
			},
			wantErr: false,
		},
		{
			name: "valid gt filter",
			filter: filter.FilterFloat{
				Gt: lo.ToPtr(42.5),
			},
			wantErr: false,
		},
		{
			name: "valid And filter",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Gt: lo.ToPtr(10.5)},
				},
			},
			wantErr: false,
		},
		{
			name: "valid Or filter",
			filter: filter.FilterFloat{
				Or: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Lt: lo.ToPtr(100.5)},
				},
			},
			wantErr: false,
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
			wantErr: false,
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
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.EqualError(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterFloat_SelectWhereExpr(t *testing.T) {
	tests := []struct {
		name      string
		filter    filter.FilterFloat
		field     string
		wantEmpty bool
		wantSQL   string
		wantArgs  []interface{}
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterFloat{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name: "eq filter",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field = ?",
			wantArgs:  []interface{}{42.5},
		},
		{
			name: "ne filter",
			filter: filter.FilterFloat{
				Ne: lo.ToPtr(42.5),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field <> ?",
			wantArgs:  []interface{}{42.5},
		},
		{
			name: "gt filter",
			filter: filter.FilterFloat{
				Gt: lo.ToPtr(42.5),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field > ?",
			wantArgs:  []interface{}{42.5},
		},
		{
			name: "and filter",
			filter: filter.FilterFloat{
				And: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Gt: lo.ToPtr(10.5)},
				},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (test_field = ? AND test_field > ?)",
			wantArgs:  []interface{}{42.5, 10.5},
		},
		{
			name: "or filter",
			filter: filter.FilterFloat{
				Or: &[]filter.FilterFloat{
					{Eq: lo.ToPtr(42.5)},
					{Lt: lo.ToPtr(100.5)},
				},
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (test_field = ? OR test_field < ?)",
			wantArgs:  []interface{}{42.5, 100.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
				return
			}

			assert.NotEmpty(t, expr, "SQL expression should not be empty")

			q.Where(expr)
			sql, args := q.Build()

			assert.Equal(t, tt.wantSQL, sql, "SQL statement should match expected value")
			assert.Equal(t, tt.wantArgs, args, "SQL arguments should match expected values")
		})
	}
}

func TestFilterBoolean_Validate(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterBoolean
		wantErr    bool
		errMessage string
	}{
		{
			name:    "nil filter",
			filter:  filter.FilterBoolean{},
			wantErr: false,
		},
		{
			name: "valid eq filter",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.EqualError(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterBoolean_SelectWhereExpr(t *testing.T) {
	tests := []struct {
		name      string
		filter    filter.FilterBoolean
		field     string
		wantEmpty bool
		wantSQL   string
		wantArgs  []interface{}
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterBoolean{},
			field:     "test_field",
			wantEmpty: true,
		},
		{
			name: "eq filter true",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(true),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field = ?",
			wantArgs:  []interface{}{true},
		},
		{
			name: "eq filter false",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(false),
			},
			field:     "test_field",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE test_field = ?",
			wantArgs:  []interface{}{false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := sqlbuilder.Select("*").From("table")
			expr := tt.filter.SelectWhereExpr(tt.field, q)

			if tt.wantEmpty {
				assert.Empty(t, expr, "SQL expression should be empty")
				return
			}

			assert.NotEmpty(t, expr, "SQL expression should not be empty")

			q.Where(expr)
			sql, args := q.Build()

			assert.Equal(t, tt.wantSQL, sql, "SQL statement should match expected value")
			assert.Equal(t, tt.wantArgs, args, "SQL arguments should match expected values")
		})
	}
}

func TestFilterTime_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  filter.FilterTime
		wantErr bool
	}{
		{
			name: "valid single filter",
			filter: filter.FilterTime{
				Gt: lo.ToPtr(time.Now()),
			},
			wantErr: false,
		},
		{
			name: "valid AND filter",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gt: lo.ToPtr(time.Now())},
					{Lt: lo.ToPtr(time.Now().Add(24 * time.Hour))},
				},
			},
			wantErr: false,
		},
		{
			name: "valid OR filter",
			filter: filter.FilterTime{
				Or: &[]filter.FilterTime{
					{Gt: lo.ToPtr(time.Now())},
					{Lt: lo.ToPtr(time.Now().Add(24 * time.Hour))},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid multiple filters",
			filter: filter.FilterTime{
				Gt:  lo.ToPtr(time.Now()),
				Gte: lo.ToPtr(time.Now()),
			},
			wantErr: true,
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterTime_SelectWhereExpr(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		filter    filter.FilterTime
		field     string
		wantEmpty bool
		wantSQL   string
		wantArgs  []interface{}
	}{
		{
			name:      "nil filter",
			filter:    filter.FilterTime{},
			field:     "created_at",
			wantEmpty: true,
		},
		{
			name: "greater than",
			filter: filter.FilterTime{
				Gt: lo.ToPtr(now),
			},
			field:     "created_at",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE created_at > ?",
			wantArgs:  []interface{}{now},
		},
		{
			name: "greater than or equal",
			filter: filter.FilterTime{
				Gte: lo.ToPtr(now),
			},
			field:     "created_at",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE created_at >= ?",
			wantArgs:  []interface{}{now},
		},
		{
			name: "less than",
			filter: filter.FilterTime{
				Lt: lo.ToPtr(now),
			},
			field:     "created_at",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE created_at < ?",
			wantArgs:  []interface{}{now},
		},
		{
			name: "less than or equal",
			filter: filter.FilterTime{
				Lte: lo.ToPtr(now),
			},
			field:     "created_at",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE created_at <= ?",
			wantArgs:  []interface{}{now},
		},
		{
			name: "AND combination",
			filter: filter.FilterTime{
				And: &[]filter.FilterTime{
					{Gt: lo.ToPtr(now)},
					{Lt: lo.ToPtr(now.Add(24 * time.Hour))},
				},
			},
			field:     "created_at",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (created_at > ? AND created_at < ?)",
			wantArgs:  []interface{}{now, now.Add(24 * time.Hour)},
		},
		{
			name: "OR combination",
			filter: filter.FilterTime{
				Or: &[]filter.FilterTime{
					{Gt: lo.ToPtr(now)},
					{Lt: lo.ToPtr(now.Add(24 * time.Hour))},
				},
			},
			field:     "created_at",
			wantEmpty: false,
			wantSQL:   "SELECT * FROM table WHERE (created_at > ? OR created_at < ?)",
			wantArgs:  []interface{}{now, now.Add(24 * time.Hour)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := sqlbuilder.NewSelectBuilder()
			builder.Select("*").From("table")

			result := tt.filter.SelectWhereExpr(tt.field, builder)

			if tt.wantEmpty {
				assert.Empty(t, result)
			} else {
				if !assert.NotEmpty(t, result) {
					return
				}

				builder.Where(result)
				sql, args := builder.Build()

				assert.Equal(t, tt.wantSQL, sql)
				assert.Equal(t, tt.wantArgs, args)
			}
		})
	}
}

func TestFilterString_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterString
		maxDepth   int
		wantErr    bool
		errMessage string
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterString{},
			maxDepth: 3,
			wantErr:  false,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
			},
			maxDepth: 3,
			wantErr:  false,
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
			wantErr:  false,
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
			wantErr:  false,
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
			wantErr:  false,
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
			maxDepth:   2,
			wantErr:    true,
			errMessage: "filter complexity exceeds maximum allowed depth",
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
			wantErr:  false,
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
			maxDepth:   2,
			wantErr:    true,
			errMessage: "filter complexity exceeds maximum allowed depth",
		},
		{
			name: "filter with validation error",
			filter: filter.FilterString{
				Eq: lo.ToPtr("test"),
				Ne: lo.ToPtr("test"),
			},
			maxDepth:   3,
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateWithComplexity(tt.maxDepth)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.ErrorContains(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterInteger_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterInteger
		maxDepth   int
		wantErr    bool
		errMessage string
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterInteger{},
			maxDepth: 3,
			wantErr:  false,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
			},
			maxDepth: 3,
			wantErr:  false,
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
			wantErr:  false,
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
			wantErr:  false,
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
			wantErr:  false,
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
			maxDepth:   2,
			wantErr:    true,
			errMessage: "filter complexity exceeds maximum allowed depth",
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
			wantErr:  false,
		},
		{
			name: "filter with validation error",
			filter: filter.FilterInteger{
				Eq: lo.ToPtr(42),
				Ne: lo.ToPtr(42),
			},
			maxDepth:   3,
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateWithComplexity(tt.maxDepth)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.ErrorContains(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterFloat_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name       string
		filter     filter.FilterFloat
		maxDepth   int
		wantErr    bool
		errMessage string
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterFloat{},
			maxDepth: 3,
			wantErr:  false,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
			},
			maxDepth: 3,
			wantErr:  false,
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
			wantErr:  false,
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
			wantErr:  false,
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
			maxDepth:   2,
			wantErr:    true,
			errMessage: "filter complexity exceeds maximum allowed depth",
		},
		{
			name: "filter with validation error",
			filter: filter.FilterFloat{
				Eq: lo.ToPtr(42.5),
				Ne: lo.ToPtr(42.5),
			},
			maxDepth:   3,
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateWithComplexity(tt.maxDepth)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.ErrorContains(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterBoolean_ValidateWithComplexity(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter.FilterBoolean
		maxDepth int
		wantErr  bool
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterBoolean{},
			maxDepth: 3,
			wantErr:  false,
		},
		{
			name: "simple filter",
			filter: filter.FilterBoolean{
				Eq: lo.ToPtr(true),
			},
			maxDepth: 3,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateWithComplexity(tt.maxDepth)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterTime_ValidateWithComplexity(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		filter     filter.FilterTime
		maxDepth   int
		wantErr    bool
		errMessage string
	}{
		{
			name:     "nil filter",
			filter:   filter.FilterTime{},
			maxDepth: 3,
			wantErr:  false,
		},
		{
			name: "simple filter within depth limit",
			filter: filter.FilterTime{
				Gt: lo.ToPtr(now),
			},
			maxDepth: 3,
			wantErr:  false,
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
			wantErr:  false,
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
			wantErr:  false,
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
			maxDepth:   2,
			wantErr:    true,
			errMessage: "filter complexity exceeds maximum allowed depth",
		},
		{
			name: "filter with validation error",
			filter: filter.FilterTime{
				Gt:  lo.ToPtr(now),
				Gte: lo.ToPtr(now),
			},
			maxDepth:   3,
			wantErr:    true,
			errMessage: "only one filter can be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateWithComplexity(tt.maxDepth)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.ErrorContains(t, err, tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
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
