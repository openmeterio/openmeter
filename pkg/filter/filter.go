package filter

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Filter is a filter for a field.
type Filter interface {
	// Validate validates the filter.
	Validate() error
	// ValidateWithComplexity validates the complexity of the filter.
	ValidateWithComplexity(maxDepth int) error
	// Select converts the filter to an Ent selector predicate.
	Select(field string) func(*sql.Selector)
	// SelectWhereExpr converts the filter to a SQL WHERE expression.
	SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string
	// IsEmpty returns true if the filter is empty.
	IsEmpty() bool
}

var (
	ErrFilterMultipleOperators  = errors.New("filter is invalid: multiple operators are set")
	ErrFilterComplexityExceeded = errors.New("filter complexity exceeds maximum allowed depth")
)

var (
	_ Filter = (*FilterString)(nil)
	_ Filter = (*FilterInteger)(nil)
	_ Filter = (*FilterFloat)(nil)
	_ Filter = (*FilterTime)(nil)
	_ Filter = (*FilterTimeUnix)(nil)
	_ Filter = (*FilterBoolean)(nil)
)

// EscapeLikePattern escapes SQL LIKE metacharacters using the package's escape character.
func EscapeLikePattern(value string) string {
	likeEscapeCharacter := `\`
	escaped := strings.ReplaceAll(value, likeEscapeCharacter, likeEscapeCharacter+likeEscapeCharacter)
	escaped = strings.ReplaceAll(escaped, "%", likeEscapeCharacter+"%")
	escaped = strings.ReplaceAll(escaped, "_", likeEscapeCharacter+"_")

	return escaped
}

// ContainsPattern builds a LIKE pattern for a literal contains-match.
func ContainsPattern(value string) string {
	return fmt.Sprintf("%%%s%%", EscapeLikePattern(value))
}

// ReverseContainsPattern extracts the plain value from a LIKE pattern
// produced by ContainsPattern (e.g. "%foo%" → "foo").
func ReverseContainsPattern(like *string) *string {
	if like == nil {
		return nil
	}
	v := *like
	v = strings.TrimPrefix(v, "%")
	v = strings.TrimSuffix(v, "%")
	v = strings.ReplaceAll(v, `\_`, "_")
	v = strings.ReplaceAll(v, `\%`, "%")
	v = strings.ReplaceAll(v, `\\`, `\`)
	return &v
}

// FilterString is a filter for a string field.
type FilterString struct {
	Eq        *string         `json:"$eq,omitempty"`
	Ne        *string         `json:"$ne,omitempty"`
	Exists    *bool           `json:"$exists,omitempty"`
	In        *[]string       `json:"$in,omitempty"`
	Nin       *[]string       `json:"$nin,omitempty"`
	Like      *string         `json:"$like,omitempty"`
	Nlike     *string         `json:"$nlike,omitempty"`
	Ilike     *string         `json:"$ilike,omitempty"`
	Nilike    *string         `json:"$nilike,omitempty"`
	Contains  *string         `json:"$contains,omitempty"`
	Ncontains *string         `json:"$ncontains,omitempty"`
	Gt        *string         `json:"$gt,omitempty"`
	Gte       *string         `json:"$gte,omitempty"`
	Lt        *string         `json:"$lt,omitempty"`
	Lte       *string         `json:"$lte,omitempty"`
	And       *[]FilterString `json:"$and,omitempty"`
	Or        *[]FilterString `json:"$or,omitempty"`
}

// Validate validates the filter.
func (f FilterString) Validate() error {
	return models.NewNillableGenericValidationError(f.validateWithComplexity(math.MaxInt))
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterString) ValidateWithComplexity(maxDepth int) error {
	return models.NewNillableGenericValidationError(f.validateWithComplexity(maxDepth))
}

// validateWithComplexity walks the filter tree returning raw sentinel errors
// so the public Validate entry points can wrap the result exactly once.
func (f FilterString) validateWithComplexity(maxDepth int) error {
	if err := validateSingleOperator(f); err != nil {
		return err
	}

	if f.And == nil && f.Or == nil {
		return nil
	}

	if maxDepth <= 0 {
		return ErrFilterComplexityExceeded
	}

	for _, child := range lo.FromPtr(f.And) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	for _, child := range lo.FromPtr(f.Or) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterString) IsEmpty() bool {
	return isEmptyFilter(f)
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterString) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, *f.Eq)
	case f.Ne != nil:
		return q.NE(field, *f.Ne)
	case f.Exists != nil:
		if *f.Exists {
			return q.IsNotNull(field)
		}
		return q.IsNull(field)
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
	case f.Contains != nil:
		return q.ILike(field, ContainsPattern(*f.Contains))
	case f.Ncontains != nil:
		return q.NotILike(field, ContainsPattern(*f.Ncontains))
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

// Select converts the filter to an Ent selector predicate.
func (f FilterString) Select(field string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	switch {
	case f.Eq != nil:
		return sql.FieldEQ(field, *f.Eq)
	case f.Ne != nil:
		return sql.FieldNEQ(field, *f.Ne)
	case f.Exists != nil:
		if *f.Exists {
			return sql.FieldNotNull(field)
		}
		return sql.FieldIsNull(field)
	case f.In != nil:
		return sql.FieldIn(field, (*f.In)...)
	case f.Nin != nil:
		return sql.FieldNotIn(field, (*f.Nin)...)
	case f.Like != nil:
		return fieldLike(field, *f.Like, false, false)
	case f.Nlike != nil:
		return fieldLike(field, *f.Nlike, true, false)
	case f.Ilike != nil:
		return fieldLike(field, *f.Ilike, false, true)
	case f.Nilike != nil:
		return fieldLike(field, *f.Nilike, true, true)
	case f.Contains != nil:
		return sql.FieldContainsFold(field, *f.Contains)
	case f.Ncontains != nil:
		return fieldLike(field, ContainsPattern(*f.Ncontains), true, true)
	case f.Gt != nil:
		return sql.FieldGT(field, *f.Gt)
	case f.Gte != nil:
		return sql.FieldGTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.FieldLT(field, *f.Lt)
	case f.Lte != nil:
		return sql.FieldLTE(field, *f.Lte)
	case f.And != nil:
		return sql.AndPredicates(lo.FilterMap(*f.And, func(filter FilterString, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	case f.Or != nil:
		return sql.OrPredicates(lo.FilterMap(*f.Or, func(filter FilterString, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	default:
		return nil
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
	return models.NewNillableGenericValidationError(f.validateWithComplexity(math.MaxInt))
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterInteger) ValidateWithComplexity(maxDepth int) error {
	return models.NewNillableGenericValidationError(f.validateWithComplexity(maxDepth))
}

// validateWithComplexity walks the filter tree returning raw sentinel errors
// and returns an error when the maximum complexity is exceeded.
func (f FilterInteger) validateWithComplexity(maxDepth int) error {
	if err := validateSingleOperator(f); err != nil {
		return err
	}

	if f.And == nil && f.Or == nil {
		return nil
	}

	if maxDepth <= 0 {
		return ErrFilterComplexityExceeded
	}

	for _, child := range lo.FromPtr(f.And) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	for _, child := range lo.FromPtr(f.Or) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterInteger) IsEmpty() bool {
	return isEmptyFilter(f)
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

// Select converts the filter to an Ent selector predicate.
func (f FilterInteger) Select(field string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	switch {
	case f.Eq != nil:
		return sql.FieldEQ(field, *f.Eq)
	case f.Ne != nil:
		return sql.FieldNEQ(field, *f.Ne)
	case f.Gt != nil:
		return sql.FieldGT(field, *f.Gt)
	case f.Gte != nil:
		return sql.FieldGTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.FieldLT(field, *f.Lt)
	case f.Lte != nil:
		return sql.FieldLTE(field, *f.Lte)
	case f.And != nil:
		return sql.AndPredicates(lo.FilterMap(*f.And, func(filter FilterInteger, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	case f.Or != nil:
		return sql.OrPredicates(lo.FilterMap(*f.Or, func(filter FilterInteger, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
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
	return models.NewNillableGenericValidationError(f.validateWithComplexity(math.MaxInt))
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterFloat) ValidateWithComplexity(maxDepth int) error {
	return models.NewNillableGenericValidationError(f.validateWithComplexity(maxDepth))
}

// validateWithComplexity walks the filter tree returning raw sentinel errors
// so the public Validate entry points can wrap the result exactly once.
func (f FilterFloat) validateWithComplexity(maxDepth int) error {
	if err := validateSingleOperator(f); err != nil {
		return err
	}

	if f.And == nil && f.Or == nil {
		return nil
	}

	if maxDepth <= 0 {
		return ErrFilterComplexityExceeded
	}

	for _, child := range lo.FromPtr(f.And) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	for _, child := range lo.FromPtr(f.Or) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterFloat) IsEmpty() bool {
	return isEmptyFilter(f)
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

// Select converts the filter to an Ent selector predicate.
func (f FilterFloat) Select(field string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	switch {
	case f.Eq != nil:
		return sql.FieldEQ(field, *f.Eq)
	case f.Ne != nil:
		return sql.FieldNEQ(field, *f.Ne)
	case f.Gt != nil:
		return sql.FieldGT(field, *f.Gt)
	case f.Gte != nil:
		return sql.FieldGTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.FieldLT(field, *f.Lt)
	case f.Lte != nil:
		return sql.FieldLTE(field, *f.Lte)
	case f.And != nil:
		return sql.AndPredicates(lo.FilterMap(*f.And, func(filter FilterFloat, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	case f.Or != nil:
		return sql.OrPredicates(lo.FilterMap(*f.Or, func(filter FilterFloat, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	default:
		return nil
	}
}

// FilterTime is a filter for a time field.
type FilterTime struct {
	Eq  *time.Time    `json:"$eq,omitempty"`
	Gt  *time.Time    `json:"$gt,omitempty"`
	Gte *time.Time    `json:"$gte,omitempty"`
	Lt  *time.Time    `json:"$lt,omitempty"`
	Lte *time.Time    `json:"$lte,omitempty"`
	And *[]FilterTime `json:"$and,omitempty"`
	Or  *[]FilterTime `json:"$or,omitempty"`
}

// Validate validates the filter.
func (f FilterTime) Validate() error {
	return models.NewNillableGenericValidationError(f.validateWithComplexity(math.MaxInt))
}

// ValidateWithComplexity validates the filter complexity.
func (f FilterTime) ValidateWithComplexity(maxDepth int) error {
	return models.NewNillableGenericValidationError(f.validateWithComplexity(maxDepth))
}

// validateWithComplexity walks the filter tree returning raw sentinel errors
// so the public Validate entry points can wrap the result exactly once.
func (f FilterTime) validateWithComplexity(maxDepth int) error {
	if err := validateSingleOperator(f); err != nil {
		return err
	}

	if f.And == nil && f.Or == nil {
		return nil
	}

	if maxDepth <= 0 {
		return ErrFilterComplexityExceeded
	}

	for _, child := range lo.FromPtr(f.And) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	for _, child := range lo.FromPtr(f.Or) {
		if err := child.validateWithComplexity(maxDepth - 1); err != nil {
			return err
		}
	}
	return nil
}

// IsEmpty returns true if the filter is empty.
func (f FilterTime) IsEmpty() bool {
	return isEmptyFilter(f)
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterTime) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, *f.Eq)
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

// Select converts the filter to an Ent selector predicate.
func (f FilterTime) Select(field string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	switch {
	case f.Eq != nil:
		return sql.FieldEQ(field, *f.Eq)
	case f.Gt != nil:
		return sql.FieldGT(field, *f.Gt)
	case f.Gte != nil:
		return sql.FieldGTE(field, *f.Gte)
	case f.Lt != nil:
		return sql.FieldLT(field, *f.Lt)
	case f.Lte != nil:
		return sql.FieldLTE(field, *f.Lte)
	case f.And != nil:
		return sql.AndPredicates(lo.FilterMap(*f.And, func(filter FilterTime, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	case f.Or != nil:
		return sql.OrPredicates(lo.FilterMap(*f.Or, func(filter FilterTime, _ int) (func(*sql.Selector), bool) {
			predicate := filter.Select(field)
			return predicate, predicate != nil
		})...)
	default:
		return nil
	}
}

// FilterTimeUnix is a filter for a time, but the generated SQL is using the
// unix timestamp in seconds.
type FilterTimeUnix struct {
	FilterTime
}

// SelectWhereExpr converts the filter to a SQL WHERE expression.
func (f FilterTimeUnix) SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string {
	switch {
	case f.Eq != nil:
		return q.EQ(field, f.Eq.Unix())
	case f.Gt != nil:
		return q.GT(field, f.Gt.Unix())
	case f.Gte != nil:
		return q.GTE(field, f.Gte.Unix())
	case f.Lt != nil:
		return q.LT(field, f.Lt.Unix())
	case f.Lte != nil:
		return q.LTE(field, f.Lte.Unix())
	case f.And != nil:
		return q.And(lo.Map(*f.And, func(filter FilterTime, _ int) string {
			return FilterTimeUnix{FilterTime: filter}.SelectWhereExpr(field, q)
		})...)
	case f.Or != nil:
		return q.Or(lo.Map(*f.Or, func(filter FilterTime, _ int) string {
			return FilterTimeUnix{FilterTime: filter}.SelectWhereExpr(field, q)
		})...)
	default:
		return ""
	}
}

// Select converts the filter to an Ent selector predicate.
func (f FilterTimeUnix) Select(field string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	switch {
	case f.Eq != nil:
		return sql.FieldEQ(field, f.Eq.Unix())
	case f.Gt != nil:
		return sql.FieldGT(field, f.Gt.Unix())
	case f.Gte != nil:
		return sql.FieldGTE(field, f.Gte.Unix())
	case f.Lt != nil:
		return sql.FieldLT(field, f.Lt.Unix())
	case f.Lte != nil:
		return sql.FieldLTE(field, f.Lte.Unix())
	case f.And != nil:
		return sql.AndPredicates(lo.FilterMap(*f.And, func(filter FilterTime, _ int) (func(*sql.Selector), bool) {
			predicate := (FilterTimeUnix{FilterTime: filter}).Select(field)
			return predicate, predicate != nil
		})...)
	case f.Or != nil:
		return sql.OrPredicates(lo.FilterMap(*f.Or, func(filter FilterTime, _ int) (func(*sql.Selector), bool) {
			predicate := (FilterTimeUnix{FilterTime: filter}).Select(field)
			return predicate, predicate != nil
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
	return isEmptyFilter(f)
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

// Select converts the filter to an Ent selector predicate.
func (f FilterBoolean) Select(field string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	switch {
	case f.Eq != nil:
		return sql.FieldEQ(field, *f.Eq)
	default:
		return nil
	}
}

// validateSingleOperator checks that at most one operator field is set on a
// filter struct. To combine operators, use the And or Or fields.
func validateSingleOperator(v Filter) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	count := 0
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		if f.Kind() == reflect.Pointer && !f.IsNil() {
			count++
		}
	}

	if count > 1 {
		return ErrFilterMultipleOperators
	}

	return nil
}

// isEmptyFilter checks that all pointer fields on a filter struct are nil.
func isEmptyFilter(v Filter) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		if f.Kind() == reflect.Pointer && !f.IsNil() {
			return false
		}
	}

	return true
}

func fieldLike(field, value string, negate, insensitive bool) func(*sql.Selector) {
	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString(s.C(field))
			if negate {
				b.WriteString(" NOT")
			}
			if insensitive {
				b.WriteString(" ILIKE ")
			} else {
				b.WriteString(" LIKE ")
			}
			b.Arg(value)
		}))
	}
}
