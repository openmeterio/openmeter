package ledger

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type RoutingKeyVersion string

const RoutingKeyVersionV1 RoutingKeyVersion = "v1"

func (v RoutingKeyVersion) Validate() error {
	switch v {
	case RoutingKeyVersionV1:
		return nil
	default:
		return ErrRoutingKeyVersionInvalid.WithAttrs(models.Attributes{
			"routing_key_version": v,
		})
	}
}

type RoutingKey struct {
	version RoutingKeyVersion
	value   string
}

func NewRoutingKey(version RoutingKeyVersion, value string) (RoutingKey, error) {
	if err := version.Validate(); err != nil {
		return RoutingKey{}, err
	}
	if value == "" {
		return RoutingKey{}, errors.New("routing key is required")
	}

	return RoutingKey{
		version: version,
		value:   value,
	}, nil
}

func (k RoutingKey) Version() RoutingKeyVersion {
	return k.version
}

func (k RoutingKey) Value() string {
	return k.value
}

func MustNewRoutingKey(version RoutingKeyVersion, value string) RoutingKey {
	key, err := NewRoutingKey(version, value)
	if err != nil {
		panic(err)
	}
	return key
}

type SubAccountRoute struct {
	id  string
	key RoutingKey
}

func NewSubAccountRoute(id string, key RoutingKey) (SubAccountRoute, error) {
	if id == "" {
		return SubAccountRoute{}, errors.New("route id is required")
	}

	return SubAccountRoute{
		id:  id,
		key: key,
	}, nil
}

func MustNewSubAccountRoute(id string, key RoutingKey) SubAccountRoute {
	route, err := NewSubAccountRoute(id, key)
	if err != nil {
		panic(err)
	}
	return route
}

func (r SubAccountRoute) ID() string {
	return r.id
}

func (r SubAccountRoute) RoutingKey() RoutingKey {
	return r.key
}

// ----------------------------------------------------------------------------
// Route — the canonical set of routing values for a sub-account
// ----------------------------------------------------------------------------

// Route holds the literal values that identify a sub-account's routing path.
// It is used for creation, persistence, and routing key generation.
type Route struct {
	Currency       currencyx.Code
	TaxCode        *string
	Features       []string
	CostBasis      *alpacadecimal.Decimal
	CreditPriority *int
}

func (r Route) Validate() error {
	if err := ValidateCurrency(r.Currency); err != nil {
		return err
	}
	if r.CreditPriority != nil {
		if err := ValidateCreditPriority(*r.CreditPriority); err != nil {
			return err
		}
	}
	if _, err := normalizeOptionalCostBasis(r.CostBasis); err != nil {
		return err
	}
	return nil
}

// Filter converts a Route to a RouteFilter for use in queries.
func (r Route) Filter() RouteFilter {
	return RouteFilter(r)
}

// Normalize canonicalizes route values so semantically equivalent routes share
// the same stored literals and routing keys.
func (r Route) Normalize() (Route, error) {
	if err := r.Validate(); err != nil {
		return Route{}, err
	}

	normalized := r
	normalized.Features = SortedFeatures(r.Features)

	costBasis, err := normalizeOptionalCostBasis(r.CostBasis)
	if err != nil {
		return Route{}, err
	}
	normalized.CostBasis = costBasis

	return normalized, nil
}

// Normalize canonicalizes route filter values before querying.
func (f RouteFilter) Normalize() (RouteFilter, error) {
	if f.Currency == "" && f.TaxCode == nil && len(f.Features) == 0 && f.CostBasis == nil && f.CreditPriority == nil {
		return f, nil
	}

	normalized, err := Route(f).Normalize()
	if err != nil {
		return RouteFilter{}, err
	}

	return RouteFilter(normalized), nil
}

// ----------------------------------------------------------------------------
// Routing key generation
// ----------------------------------------------------------------------------

func BuildRoutingKey(version RoutingKeyVersion, route Route) (RoutingKey, error) {
	if err := route.Validate(); err != nil {
		return RoutingKey{}, err
	}

	switch version {
	case RoutingKeyVersionV1:
		return BuildRoutingKeyV1(route)
	default:
		return RoutingKey{}, ErrRoutingKeyVersionUnsupported.WithAttrs(models.Attributes{
			"routing_key_version": version,
		})
	}
}

func BuildRoutingKeyV1(route Route) (RoutingKey, error) {
	normalizedRoute, err := route.Normalize()
	if err != nil {
		return RoutingKey{}, err
	}

	value := strings.Join([]string{
		"currency:" + string(normalizedRoute.Currency),
		"tax_code:" + optionalStringValue(normalizedRoute.TaxCode),
		"features:" + canonicalFeatures(normalizedRoute.Features),
		"cost_basis:" + optionalDecimalValue(normalizedRoute.CostBasis),
		"credit_priority:" + optionalIntValue(normalizedRoute.CreditPriority),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV1, value)
}

// ----------------------------------------------------------------------------
// Validation helpers
// ----------------------------------------------------------------------------

// ValidateCreditPriority validates a credit priority integer value.
func ValidateCreditPriority(value int) error {
	if value < 1 {
		return ErrCreditPriorityInvalid.WithAttrs(models.Attributes{
			"credit_priority": value,
		})
	}
	return nil
}

// ValidateCurrency validates a currency value.
func ValidateCurrency(value currencyx.Code) error {
	if err := value.Validate(); err != nil {
		return ErrCurrencyInvalid.WithAttrs(models.Attributes{
			"currency": value,
		})
	}

	return nil
}

// ----------------------------------------------------------------------------
// internal helpers
// ----------------------------------------------------------------------------

// SortedFeatures returns a sorted copy of features for canonical storage.
// Returns nil if empty.
func SortedFeatures(features []string) []string {
	if len(features) == 0 {
		return nil
	}
	sorted := make([]string, len(features))
	copy(sorted, features)
	sort.Strings(sorted)
	return sorted
}

func canonicalFeatures(features []string) string {
	if len(features) == 0 {
		return "null"
	}

	return strings.Join(SortedFeatures(features), ",")
}

func optionalStringValue(s *string) string {
	if s == nil || *s == "" {
		return "null"
	}
	return *s
}

func optionalIntValue(v *int) string {
	if v == nil {
		return "null"
	}
	return strconv.Itoa(*v)
}

func optionalDecimalValue(v *alpacadecimal.Decimal) string {
	if v == nil {
		return "null"
	}
	return v.String()
}

func normalizeOptionalCostBasis(v *alpacadecimal.Decimal) (*alpacadecimal.Decimal, error) {
	if v == nil {
		return nil, nil
	}

	if v.IsNegative() {
		return nil, ErrCostBasisInvalid.WithAttrs(models.Attributes{
			"cost_basis": v.String(),
		})
	}

	normalized, err := alpacadecimal.NewFromString(v.String())
	if err != nil {
		return nil, fmt.Errorf("normalize cost basis: %w", err)
	}

	return &normalized, nil
}
