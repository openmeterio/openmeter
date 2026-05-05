package ledger

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type RoutingKeyVersion string

const RoutingKeyVersionV1 RoutingKeyVersion = "v1"

type TransactionAuthorizationStatus string

const (
	TransactionAuthorizationStatusOpen       TransactionAuthorizationStatus = "open"
	TransactionAuthorizationStatusAuthorized TransactionAuthorizationStatus = "authorized"
)

type TaxBehavior string

const (
	TaxBehaviorInclusive TaxBehavior = "inclusive"
	TaxBehaviorExclusive TaxBehavior = "exclusive"
)

func (s TransactionAuthorizationStatus) Validate() error {
	switch s {
	case TransactionAuthorizationStatusOpen:
		return nil
	case TransactionAuthorizationStatusAuthorized:
		return nil
	default:
		return ErrTransactionAuthorizationStatusInvalid.WithAttrs(models.Attributes{
			"transaction_authorization_status": s,
		})
	}
}

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

func (b TaxBehavior) Validate() error {
	switch b {
	case TaxBehaviorInclusive:
		return nil
	case TaxBehaviorExclusive:
		return nil
	default:
		return ErrTaxBehaviorInvalid.WithAttrs(models.Attributes{
			"tax_behavior": b,
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

type SubAccountRoute struct {
	id    string
	key   RoutingKey
	route Route
}

// NewSubAccountRouteFromData hydrates a sub-account route from persisted data.
// Does not enforce Route & RoutingKey equality due to possible version mismatch
func NewSubAccountRouteFromData(id string, key RoutingKey, route Route) (SubAccountRoute, error) {
	if id == "" {
		return SubAccountRoute{}, errors.New("route id is required")
	}
	if err := route.Validate(); err != nil {
		return SubAccountRoute{}, fmt.Errorf("route: %w", err)
	}

	normalizedRoute, err := route.Normalize()
	if err != nil {
		return SubAccountRoute{}, fmt.Errorf("normalize route: %w", err)
	}

	return SubAccountRoute{
		id:    id,
		key:   key,
		route: normalizedRoute,
	}, nil
}

// NewSubAccountRouteFromRoute creates a new sub-account route from a literal route
func NewSubAccountRouteFromRoute(id string, version RoutingKeyVersion, route Route) (SubAccountRoute, error) {
	if id == "" {
		return SubAccountRoute{}, errors.New("route id is required")
	}
	if err := route.Validate(); err != nil {
		return SubAccountRoute{}, fmt.Errorf("route: %w", err)
	}

	normalizedRoute, err := route.Normalize()
	if err != nil {
		return SubAccountRoute{}, fmt.Errorf("normalize route: %w", err)
	}

	key, err := BuildRoutingKey(version, normalizedRoute)
	if err != nil {
		return SubAccountRoute{}, fmt.Errorf("build routing key from route: %w", err)
	}

	return SubAccountRoute{
		id:    id,
		key:   key,
		route: normalizedRoute,
	}, nil
}

func (r SubAccountRoute) ID() string {
	return r.id
}

func (r SubAccountRoute) RoutingKey() RoutingKey {
	return r.key
}

func (r SubAccountRoute) Route() Route {
	return r.route
}

// ----------------------------------------------------------------------------
// Route — the canonical set of routing values for a sub-account
// ----------------------------------------------------------------------------

// Route holds the literal values that identify a sub-account's routing path.
// It is used for creation, persistence, and routing key generation.
type Route struct {
	Currency                       currencyx.Code
	TaxCode                        *string
	TaxBehavior                    *TaxBehavior
	Features                       []string
	CostBasis                      *alpacadecimal.Decimal
	CreditPriority                 *int
	TransactionAuthorizationStatus *TransactionAuthorizationStatus
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

	if r.CostBasis != nil {
		if err := ValidateCostBasis(*r.CostBasis); err != nil {
			return err
		}
	}

	if r.TransactionAuthorizationStatus != nil {
		if err := r.TransactionAuthorizationStatus.Validate(); err != nil {
			return err
		}
	}

	if r.TaxBehavior != nil {
		if err := r.TaxBehavior.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Filter converts a Route to a RouteFilter for use in queries.
// All present route fields are pinned as exact-match filters (including nil values).
func (r Route) Filter() RouteFilter {
	return RouteFilter{
		Currency:                       r.Currency,
		TaxCode:                        mo.Some(r.TaxCode),
		TaxBehavior:                    mo.Some(r.TaxBehavior),
		Features:                       r.Features,
		CostBasis:                      mo.Some(r.CostBasis),
		CreditPriority:                 r.CreditPriority,
		TransactionAuthorizationStatus: r.TransactionAuthorizationStatus,
	}
}

// Normalize canonicalizes route values so semantically equivalent routes share
// the same stored literals and routing keys.
func (r Route) Normalize() (Route, error) {
	if err := r.Validate(); err != nil {
		return Route{}, err
	}

	normalized := r
	normalized.Features = SortedFeatures(r.Features)

	return normalized, nil
}

// Normalize canonicalizes route filter values before querying.
func (f RouteFilter) Normalize() (RouteFilter, error) {
	if f.Currency == "" && f.TaxCode.IsAbsent() && len(f.Features) == 0 && f.CostBasis.IsAbsent() && f.CreditPriority == nil && f.TransactionAuthorizationStatus == nil && f.TaxBehavior.IsAbsent() {
		return f, nil
	}

	taxCode, _ := f.TaxCode.Get()
	taxBehavior, _ := f.TaxBehavior.Get()
	costBasis, _ := f.CostBasis.Get()
	normalized, err := Route{
		Currency:                       f.Currency,
		TaxCode:                        taxCode,
		TaxBehavior:                    taxBehavior,
		Features:                       f.Features,
		CostBasis:                      costBasis,
		CreditPriority:                 f.CreditPriority,
		TransactionAuthorizationStatus: f.TransactionAuthorizationStatus,
	}.Normalize()
	if err != nil {
		return RouteFilter{}, err
	}

	normalizedCostBasis := mo.None[*alpacadecimal.Decimal]()
	if f.CostBasis.IsPresent() {
		normalizedCostBasis = mo.Some(normalized.CostBasis)
	}

	normalizedTaxCode := mo.None[*string]()
	if f.TaxCode.IsPresent() {
		normalizedTaxCode = mo.Some(normalized.TaxCode)
	}

	normalizedTaxBehavior := mo.None[*TaxBehavior]()
	if f.TaxBehavior.IsPresent() {
		normalizedTaxBehavior = mo.Some(normalized.TaxBehavior)
	}

	return RouteFilter{
		Currency:                       normalized.Currency,
		TaxCode:                        normalizedTaxCode,
		TaxBehavior:                    normalizedTaxBehavior,
		Features:                       normalized.Features,
		CostBasis:                      normalizedCostBasis,
		CreditPriority:                 normalized.CreditPriority,
		TransactionAuthorizationStatus: normalized.TransactionAuthorizationStatus,
	}, nil
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
		"tax_behavior:" + optionalTaxBehaviorValue(normalizedRoute.TaxBehavior),
		"features:" + canonicalFeatures(normalizedRoute.Features),
		"cost_basis:" + optionalDecimalValue(normalizedRoute.CostBasis),
		"credit_priority:" + optionalIntValue(normalizedRoute.CreditPriority),
		"transaction_authorization_status:" + optionalTransactionAuthorizationStatusValue(normalizedRoute.TransactionAuthorizationStatus),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV1, value)
}

// ----------------------------------------------------------------------------
// Validation helpers
// ----------------------------------------------------------------------------

func ValidateTransactionAmount(value alpacadecimal.Decimal) error {
	if value.IsNegative() {
		return ErrTransactionAmountInvalid.WithAttrs(models.Attributes{
			"transaction_amount": value.String(),
		})
	}

	if value.IsZero() {
		return ErrTransactionAmountInvalid.WithAttrs(models.Attributes{
			"transaction_amount": value.String(),
		})
	}

	return nil
}

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

func ValidateCostBasis(value alpacadecimal.Decimal) error {
	if value.IsNegative() {
		return ErrCostBasisInvalid.WithAttrs(models.Attributes{
			"cost_basis": value.String(),
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

func optionalTransactionAuthorizationStatusValue(v *TransactionAuthorizationStatus) string {
	if v == nil {
		return "null"
	}
	return string(*v)
}

func optionalDecimalValue(v *alpacadecimal.Decimal) string {
	if v == nil {
		return "null"
	}
	return v.String()
}

func optionalTaxBehaviorValue(v *TaxBehavior) string {
	if v == nil {
		return "null"
	}
	return string(*v)
}
