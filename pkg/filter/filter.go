package filter

import (
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"
)

// Filter is a filter for a field.
type Filter interface {
	// Validate validates the filter.
	Validate() error
	// ValidateWithComplexity validates the complexity of the filter.
	ValidateWithComplexity(maxDepth int) error
	// SelectWhereExpr converts the filter to a SQL WHERE expression.
	SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string
	// SelectWherePredicate converts the filter to a predicate for ent.Query.Where.
	SelectWherePredicate(field string) *sql.Predicate
	// IsEmpty returns true if the filter is empty.
	IsEmpty() bool
}

var (
	_ Filter = (*FilterString)(nil)
	_ Filter = (*FilterInteger)(nil)
	_ Filter = (*FilterFloat)(nil)
	_ Filter = (*FilterTime)(nil)
	_ Filter = (*FilterBoolean)(nil)
)

// FilterString is a filter for a string field.
type FilterString struct {
	Eq     *string         `json:"$eq,omitempty"`
	Ne     *string         `json:"$ne,omitempty"`
	In     *[]string       `json:"$in,omitempty"`
	Nin    *[]string       `json:"$nin,omitempty"`
	Like   *string         `json:"$like,omitempty"`
	Nlike  *string         `json:"$nlike,omitempty"`
	Ilike  *string         `json:"$ilike,omitempty"`
	Nilike *string         `json:"$nilike,omitempty"`
	Gt     *string         `json:"$gt,omitempty"`
	Gte    *string         `json:"$gte,omitempty"`
	Lt     *string         `json:"$lt,omitempty"`
	Lte    *string         `json:"$lte,omitempty"`
	And    *[]FilterString `json:"$and,omitempty"`
	Or     *[]FilterString `json:"$or,omitempty"`
}

// Validate validates the filter.
func (f FilterString) Validate() error {
	// Check for multiple non-nil filters
	if err := validateMutuallyExclusiveFilters([]bool{
		f.Eq != nil, f.Ne != nil, f.In != nil, f.Nin != nil,
		f.Like != nil, f.Nlike != nil, f.Ilike != nil, f.Nilike != nil,
		f.Gt != nil, f.Gte != nil, f.Lt != nil, f.Lte != nil,
		f.And != nil, f.Or != nil,
	}); err != nil {
		return err
	}

	// Validate logical operators
	errs := lo.Map(lo.FromPtr(f.And), func(f FilterString, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	errs = lo.Map(lo.FromPtr(f.Or), func(f FilterString, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterString) ValidateWithComplexity(maxDepth int) error {
	// First validate the filter itself
	if err := f.Validate(); err != nil {
		return err
	}

	// Check if we're at a logical operator and need to validate the depth
	if f.And != nil || f.Or != nil {
		if maxDepth <= 0 {
			return errors.New("filter complexity exceeds maximum allowed depth")
		}

		// Validate nested filters with decremented depth
		if f.And != nil {
			errs := lo.Map(lo.FromPtr(f.And), func(f FilterString, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}

		if f.Or != nil {
			errs := lo.Map(lo.FromPtr(f.Or), func(f FilterString, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterString) IsEmpty() bool {
	return f.Eq == nil && f.Ne == nil && f.In == nil && f.Nin == nil && f.Like == nil && f.Nlike == nil && f.Ilike == nil && f.Nilike == nil && f.Gt == nil && f.Gte == nil && f.Lt == nil && f.Lte == nil && f.And == nil && f.Or == nil
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterString) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, *f.Eq)
	case f.Ne != nil:
		return q.NE(field, *f.Ne)
	case f.In != nil:
		return q.In(field, *f.In)
	case f.Nin != nil:
		return q.NotIn(field, *f.Nin)
	case f.Like != nil:
		return q.Like(field, *f.Like)
	case f.Nlike != nil:
		return q.NotLike(field, *f.Nlike)
	case f.Ilike != nil:
		return q.ILike(field, *f.Ilike)
	case f.Nilike != nil:
		return q.NotILike(field, *f.Nilike)
	case f.Gt != nil:
		return q.GT(field, *f.Gt)
	case f.Gte != nil:
		return q.GTE(field, *f.Gte)
	case f.Lt != nil:
		return q.LT(field, *f.Lt)
	case f.Lte != nil:
		return q.LTE(field, *f.Lte)
	case f.And != nil:
		return q.And(lo.Map(*f.And, func(filter FilterString, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	case f.Or != nil:
		return q.Or(lo.Map(*f.Or, func(filter FilterString, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	default:
		return ""
	}
}

// SelectWherePredicate converts the filter to an ent *sql.Predicate.
func (f FilterString) SelectWherePredicate(field string) *sql.Predicate {
	switch {
	case f.Eq != nil:
		return sql.EQ(field, *f.Eq)
	case f.Ne != nil:
		return sql.NEQ(field, *f.Ne)
	case f.In != nil:
		// We must cast *[]string to []interface{} for sql.In
		vals := lo.ToAnySlice(*f.In)
		return sql.In(field, vals...)
	case f.Nin != nil:
		vals := lo.ToAnySlice(*f.Nin)
		return sql.NotIn(field, vals...)
	case f.Like != nil:
		return sql.Like(field, *f.Like)
	case f.Nlike != nil:
		return sql.Not(sql.Like(field, *f.Nlike))
	case f.Ilike != nil:
		// Use ContainsFold for case-insensitive substring matching
		// This generates ILIKE with % wildcards automatically
		return sql.ContainsFold(field, *f.Ilike)
	case f.Nilike != nil:
		// Use NOT ContainsFold for negated case-insensitive substring matching
		return sql.Not(sql.ContainsFold(field, *f.Nilike))
	case f.Gt != nil:
		return sql.GT(field, *f.Gt)
	case f.Gte != nil:
		return sql.GTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.LT(field, *f.Lt)
	case f.Lte != nil:
		return sql.LTE(field, *f.Lte)
	case f.And != nil:
		// Recursively map the children
		preds := lo.Map(*f.And, func(sub FilterString, _ int) *sql.Predicate {
			return sub.SelectWherePredicate(field)
		})
		return sql.And(preds...)
	case f.Or != nil:
		preds := lo.Map(*f.Or, func(sub FilterString, _ int) *sql.Predicate {
			return sub.SelectWherePredicate(field)
		})
		return sql.Or(preds...)
	default:
		// No filter applied, return "always true" or nil
		return nil
	}
}

func (f FilterString) Where(colName string) func(*sql.Selector) {
	return func(s *sql.Selector) {
		p := f.SelectWherePredicate(s.C(colName))
		if p != nil {
			s.Where(p)
		}
	}
}

// FilterInteger is a filter for an integer field.
type FilterInteger struct {
	Eq  *int             `json:"$eq,omitempty"`
	Ne  *int             `json:"$ne,omitempty"`
	Gt  *int             `json:"$gt,omitempty"`
	Gte *int             `json:"$gte,omitempty"`
	Lt  *int             `json:"$lt,omitempty"`
	Lte *int             `json:"$lte,omitempty"`
	And *[]FilterInteger `json:"$and,omitempty"`
	Or  *[]FilterInteger `json:"$or,omitempty"`
}

// Validate validates the filter.
func (f FilterInteger) Validate() error {
	// Check for multiple non-nil filters
	if err := validateMutuallyExclusiveFilters([]bool{
		f.Eq != nil, f.Ne != nil, f.Gt != nil, f.Gte != nil, f.Lt != nil, f.Lte != nil,
		f.And != nil, f.Or != nil,
	}); err != nil {
		return err
	}

	// Validate logical operators
	errs := lo.Map(lo.FromPtr(f.And), func(f FilterInteger, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	errs = lo.Map(lo.FromPtr(f.Or), func(f FilterInteger, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterInteger) ValidateWithComplexity(maxDepth int) error {
	// First validate the filter itself
	if err := f.Validate(); err != nil {
		return err
	}

	// Check if we're at a logical operator and need to validate the depth
	if f.And != nil || f.Or != nil {
		if maxDepth <= 0 {
			return errors.New("filter complexity exceeds maximum allowed depth")
		}

		// Validate nested filters with decremented depth
		if f.And != nil {
			errs := lo.Map(lo.FromPtr(f.And), func(f FilterInteger, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}

		if f.Or != nil {
			errs := lo.Map(lo.FromPtr(f.Or), func(f FilterInteger, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterInteger) IsEmpty() bool {
	return f.Eq == nil && f.Ne == nil && f.Gt == nil && f.Gte == nil && f.Lt == nil && f.Lte == nil && f.And == nil && f.Or == nil
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterInteger) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, *f.Eq)
	case f.Ne != nil:
		return q.NE(field, *f.Ne)
	case f.Gt != nil:
		return q.GT(field, *f.Gt)
	case f.Gte != nil:
		return q.GTE(field, *f.Gte)
	case f.Lt != nil:
		return q.LT(field, *f.Lt)
	case f.Lte != nil:
		return q.LTE(field, *f.Lte)
	case f.And != nil:
		return q.And(lo.Map(*f.And, func(filter FilterInteger, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	case f.Or != nil:
		return q.Or(lo.Map(*f.Or, func(filter FilterInteger, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	default:
		return ""
	}
}

func (f FilterInteger) SelectWherePredicate(field string) *sql.Predicate {
	switch {
	case f.Eq != nil:
		return sql.EQ(field, *f.Eq)
	case f.Ne != nil:
		return sql.NEQ(field, *f.Ne)
	case f.Gt != nil:
		return sql.GT(field, *f.Gt)
	case f.Gte != nil:
		return sql.GTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.LT(field, *f.Lt)
	case f.Lte != nil:
		return sql.LTE(field, *f.Lte)
	case f.And != nil:
		return sql.And(lo.Map(*f.And, func(filter FilterInteger, _ int) *sql.Predicate {
			return filter.SelectWherePredicate(field)
		})...)
	case f.Or != nil:
		return sql.Or(lo.Map(*f.Or, func(filter FilterInteger, _ int) *sql.Predicate {
			return filter.SelectWherePredicate(field)
		})...)
	default:
		return nil
	}
}

// FilterFloat is a filter for a float field.
type FilterFloat struct {
	Eq  *float64       `json:"$eq,omitempty"`
	Ne  *float64       `json:"$ne,omitempty"`
	Gt  *float64       `json:"$gt,omitempty"`
	Gte *float64       `json:"$gte,omitempty"`
	Lt  *float64       `json:"$lt,omitempty"`
	Lte *float64       `json:"$lte,omitempty"`
	And *[]FilterFloat `json:"$and,omitempty"`
	Or  *[]FilterFloat `json:"$or,omitempty"`
}

func (f FilterFloat) Validate() error {
	// Check for multiple non-nil filters
	if err := validateMutuallyExclusiveFilters([]bool{
		f.Eq != nil, f.Ne != nil, f.Gt != nil, f.Gte != nil, f.Lt != nil, f.Lte != nil,
		f.And != nil, f.Or != nil,
	}); err != nil {
		return err
	}

	// Validate logical operators
	errs := lo.Map(lo.FromPtr(f.And), func(f FilterFloat, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	errs = lo.Map(lo.FromPtr(f.Or), func(f FilterFloat, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterFloat) ValidateWithComplexity(maxDepth int) error {
	// First validate the filter itself
	if err := f.Validate(); err != nil {
		return err
	}

	// Check if we're at a logical operator and need to validate the depth
	if f.And != nil || f.Or != nil {
		if maxDepth <= 0 {
			return errors.New("filter complexity exceeds maximum allowed depth")
		}

		// Validate nested filters with decremented depth
		if f.And != nil {
			errs := lo.Map(lo.FromPtr(f.And), func(f FilterFloat, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}

		if f.Or != nil {
			errs := lo.Map(lo.FromPtr(f.Or), func(f FilterFloat, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterFloat) IsEmpty() bool {
	return f.Eq == nil && f.Ne == nil && f.Gt == nil && f.Gte == nil && f.Lt == nil && f.Lte == nil && f.And == nil && f.Or == nil
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterFloat) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, *f.Eq)
	case f.Ne != nil:
		return q.NE(field, *f.Ne)
	case f.Gt != nil:
		return q.GT(field, *f.Gt)
	case f.Gte != nil:
		return q.GTE(field, *f.Gte)
	case f.Lt != nil:
		return q.LT(field, *f.Lt)
	case f.Lte != nil:
		return q.LTE(field, *f.Lte)
	case f.And != nil:
		return q.And(lo.Map(*f.And, func(filter FilterFloat, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	case f.Or != nil:
		return q.Or(lo.Map(*f.Or, func(filter FilterFloat, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	default:
		return ""
	}
}

func (f FilterFloat) SelectWherePredicate(field string) *sql.Predicate {
	switch {
	case f.Eq != nil:
		return sql.EQ(field, *f.Eq)
	case f.Ne != nil:
		return sql.NEQ(field, *f.Ne)
	case f.Gt != nil:
		return sql.GT(field, *f.Gt)
	case f.Gte != nil:
		return sql.GTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.LT(field, *f.Lt)
	case f.Lte != nil:
		return sql.LTE(field, *f.Lte)
	case f.And != nil:
		return sql.And(lo.Map(*f.And, func(filter FilterFloat, _ int) *sql.Predicate {
			return filter.SelectWherePredicate(field)
		})...)
	case f.Or != nil:
		return sql.Or(lo.Map(*f.Or, func(filter FilterFloat, _ int) *sql.Predicate {
			return filter.SelectWherePredicate(field)
		})...)
	default:
		return nil
	}
}

// FilterTime is a filter for a time field.
type FilterTime struct {
	Gt  *time.Time    `json:"$gt,omitempty"`
	Gte *time.Time    `json:"$gte,omitempty"`
	Lt  *time.Time    `json:"$lt,omitempty"`
	Lte *time.Time    `json:"$lte,omitempty"`
	And *[]FilterTime `json:"$and,omitempty"`
	Or  *[]FilterTime `json:"$or,omitempty"`
}

// Validate validates the filter.
func (f FilterTime) Validate() error {
	// Check for multiple non-nil filters
	if err := validateMutuallyExclusiveFilters([]bool{
		f.Gt != nil, f.Gte != nil, f.Lt != nil, f.Lte != nil,
		f.And != nil, f.Or != nil,
	}); err != nil {
		return err
	}

	// Validate logical operators
	errs := lo.Map(lo.FromPtr(f.And), func(f FilterTime, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	errs = lo.Map(lo.FromPtr(f.Or), func(f FilterTime, _ int) error {
		return f.Validate()
	})
	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterTime) ValidateWithComplexity(maxDepth int) error {
	// First validate the filter itself
	if err := f.Validate(); err != nil {
		return err
	}

	// Check if we're at a logical operator and need to validate the depth
	if f.And != nil || f.Or != nil {
		if maxDepth <= 0 {
			return errors.New("filter complexity exceeds maximum allowed depth")
		}

		// Validate nested filters with decremented depth
		if f.And != nil {
			errs := lo.Map(lo.FromPtr(f.And), func(f FilterTime, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}

		if f.Or != nil {
			errs := lo.Map(lo.FromPtr(f.Or), func(f FilterTime, _ int) error {
				return f.ValidateWithComplexity(maxDepth - 1)
			})
			if err := errors.Join(errs...); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterTime) IsEmpty() bool {
	return f.Gt == nil && f.Gte == nil && f.Lt == nil && f.Lte == nil && f.And == nil && f.Or == nil
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterTime) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Gt != nil:
		return q.GT(field, *f.Gt)
	case f.Gte != nil:
		return q.GTE(field, *f.Gte)
	case f.Lt != nil:
		return q.LT(field, *f.Lt)
	case f.Lte != nil:
		return q.LTE(field, *f.Lte)
	case f.And != nil:
		return q.And(lo.Map(*f.And, func(filter FilterTime, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	case f.Or != nil:
		return q.Or(lo.Map(*f.Or, func(filter FilterTime, _ int) string {
			return filter.SelectWhereExpr(field, q)
		})...)
	default:
		return ""
	}
}

func (f FilterTime) SelectWherePredicate(field string) *sql.Predicate {
	switch {
	case f.Gt != nil:
		return sql.GT(field, *f.Gt)
	case f.Gte != nil:
		return sql.GTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.LT(field, *f.Lt)
	case f.Lte != nil:
		return sql.LTE(field, *f.Lte)
	case f.And != nil:
		return sql.And(lo.Map(*f.And, func(filter FilterTime, _ int) *sql.Predicate {
			return filter.SelectWherePredicate(field)
		})...)
	case f.Or != nil:
		return sql.Or(lo.Map(*f.Or, func(filter FilterTime, _ int) *sql.Predicate {
			return filter.SelectWherePredicate(field)
		})...)
	default:
		return nil
	}
}

// FilterBoolean is a filter for a boolean field.
type FilterBoolean struct {
	Eq *bool `json:"$eq,omitempty"`
}

// Validate validates the filter.
func (f FilterBoolean) Validate() error {
	return nil
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterBoolean) ValidateWithComplexity(maxDepth int) error {
	// Boolean filter has no nested filters, so just validate normally
	return f.Validate()
}

// IsEmpty returns true if the filter is empty.
func (f FilterBoolean) IsEmpty() bool {
	return f.Eq == nil
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterBoolean) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, *f.Eq)
	default:
		return ""
	}
}

func (f FilterBoolean) SelectWherePredicate(field string) *sql.Predicate {
	switch {
	case f.Eq != nil:
		return sql.EQ(field, *f.Eq)
	default:
		return nil
	}
}

// validateMutuallyExclusiveFilters checks if more than one filter field is set, as filters are mutually exclusive
func validateMutuallyExclusiveFilters(fields []bool) error {
	nonNilFilters := lo.CountBy(fields, func(b bool) bool { return b })
	if nonNilFilters > 1 {
		return errors.New("only one filter can be set")
	}
	return nil
}
