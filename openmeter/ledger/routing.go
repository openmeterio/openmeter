package ledger

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type RoutingKeyVersion string

const RoutingKeyVersionV1 RoutingKeyVersion = "v1"

func (v RoutingKeyVersion) Validate() error {
	switch v {
	case RoutingKeyVersionV1:
		return nil
	default:
		return fmt.Errorf("invalid routing key version: %s", v)
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
	Currency       string
	TaxCode        *string
	Features       []string
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
	return nil
}

// Filter converts a Route to a RouteFilter for use in queries.
func (r Route) Filter() RouteFilter {
	return RouteFilter(r)
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
		return RoutingKey{}, fmt.Errorf("unsupported routing key version: %s", version)
	}
}

func BuildRoutingKeyV1(route Route) (RoutingKey, error) {
	if err := route.Validate(); err != nil {
		return RoutingKey{}, err
	}

	value := strings.Join([]string{
		"currency:" + route.Currency,
		"tax_code:" + optionalStringValue(route.TaxCode),
		"features:" + canonicalFeatures(route.Features),
		"credit_priority:" + optionalIntValue(route.CreditPriority),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV1, value)
}

// ----------------------------------------------------------------------------
// Validation helpers
// ----------------------------------------------------------------------------

// ValidateCreditPriority validates a credit priority integer value.
func ValidateCreditPriority(value int) error {
	if value < 1 {
		return fmt.Errorf("credit priority must be a positive integer")
	}
	return nil
}

// ValidateCurrency validates a currency string.
func ValidateCurrency(value string) error {
	if value == "" {
		return fmt.Errorf("currency is required")
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
