package ledger

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/alpacahq/alpacadecimal"
	goblcurrency "github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type RoutingKeyVersion string

const (
	RoutingKeyVersionV1 RoutingKeyVersion = "v1"
	// RoutingKeyVersionV2 extends V1 by adding the tax_behavior segment.
	// Use V2 when a route has a non-nil TaxBehavior; otherwise use V1 for
	// backward compatibility with sub-accounts created before tax_behavior existed.
	RoutingKeyVersionV2 RoutingKeyVersion = "v2"
	// RoutingKeyVersionV3 extends V2 by adding the exchange_source_currency segment.
	RoutingKeyVersionV3 RoutingKeyVersion = "v3"
)

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
	case RoutingKeyVersionV1, RoutingKeyVersionV2, RoutingKeyVersionV3:
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

type SubAccountRouteData struct {
	ID         string
	RoutingKey RoutingKey
	Route      Route
}

// NewSubAccountRouteFromData hydrates a sub-account route from persisted data.
// Does not enforce Route & RoutingKey equality due to possible version mismatch.
// Sets route.Version from the stored routing key so the round-trip is transparent.
func NewSubAccountRouteFromData(data SubAccountRouteData) (SubAccountRoute, error) {
	if data.ID == "" {
		return SubAccountRoute{}, errors.New("route id is required")
	}
	if err := data.Route.Validate(); err != nil {
		return SubAccountRoute{}, fmt.Errorf("route: %w", err)
	}

	normalizedRoute, err := data.Route.Normalize()
	if err != nil {
		return SubAccountRoute{}, fmt.Errorf("normalize route: %w", err)
	}

	if data.RoutingKey.Version() == RoutingKeyVersionV3 && normalizedRoute.ExchangeSourceCurrency == nil {
		return SubAccountRoute{}, errors.New("routing key version v3 requires exchange source currency")
	}

	normalizedRoute.Version = data.RoutingKey.Version()

	return SubAccountRoute{
		id:    data.ID,
		key:   data.RoutingKey,
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
	// Version is auto-derived by Normalize() based on which fields are present.
	// Direct assignment is overwritten on the next Normalize call. Do not set
	// manually except for testing edge cases.
	Version  RoutingKeyVersion
	Currency currencyx.Code
	// ExchangeSourceCurrency identifies the fiat currency exchanged into a custom currency.
	// Fiat and non-exchange routes keep ExchangeSourceCurrency nil.
	ExchangeSourceCurrency *currencyx.Code
	TaxCode                *string
	// TaxBehavior distinguishes taxable accrued and earnings buckets.
	// Customer FBO routes do not carry tax dimensions; credit sources are
	// attributed to charge tax configuration when they accrue.
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
	if err := ValidateExchangeSourceCurrency(r.Currency, r.ExchangeSourceCurrency); err != nil {
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

	if err := validateFeatures(r.Features); err != nil {
		return fmt.Errorf("features: %w", err)
	}

	return nil
}

// Filter converts a Route to a RouteFilter for use in queries.
// All present route fields are pinned as exact-match filters (including nil values).
func (r Route) Filter() RouteFilter {
	return RouteFilter{
		Currency:                       r.Currency,
		ExchangeSourceCurrency:         mo.Some(r.ExchangeSourceCurrency),
		TaxCode:                        mo.Some(r.TaxCode),
		TaxBehavior:                    mo.Some(r.TaxBehavior),
		Features:                       mo.Some(r.Features),
		CostBasis:                      mo.Some(r.CostBasis),
		CreditPriority:                 r.CreditPriority,
		TransactionAuthorizationStatus: r.TransactionAuthorizationStatus,
	}
}

func (r Route) Matches(filter RouteFilter) bool {
	if filter.Currency != "" && r.Currency != filter.Currency {
		return false
	}
	if filter.ExchangeSourceCurrency.IsPresent() {
		exchangeSourceCurrency, _ := filter.ExchangeSourceCurrency.Get()
		switch {
		case exchangeSourceCurrency == nil && r.ExchangeSourceCurrency != nil:
			return false
		case exchangeSourceCurrency != nil && r.ExchangeSourceCurrency == nil:
			return false
		case exchangeSourceCurrency != nil && r.ExchangeSourceCurrency != nil && *exchangeSourceCurrency != *r.ExchangeSourceCurrency:
			return false
		}
	}
	if filter.TaxCode.IsPresent() {
		taxCode, _ := filter.TaxCode.Get()
		switch {
		case taxCode == nil && r.TaxCode != nil:
			return false
		case taxCode != nil && r.TaxCode == nil:
			return false
		case taxCode != nil && r.TaxCode != nil && *taxCode != *r.TaxCode:
			return false
		}
	}
	if filter.TaxBehavior.IsPresent() {
		taxBehavior, _ := filter.TaxBehavior.Get()
		switch {
		case taxBehavior == nil && r.TaxBehavior != nil:
			return false
		case taxBehavior != nil && r.TaxBehavior == nil:
			return false
		case taxBehavior != nil && r.TaxBehavior != nil && *taxBehavior != *r.TaxBehavior:
			return false
		}
	}
	if filter.Features.IsPresent() {
		features, _ := filter.Features.Get()
		if !slices.Equal(SortedFeatures(r.Features), SortedFeatures(features)) {
			return false
		}
	}
	if filter.MatchFeature != "" {
		if len(r.Features) > 0 && !slices.Contains(r.Features, filter.MatchFeature) {
			return false
		}
	}
	if filter.CostBasis.IsPresent() {
		costBasis, _ := filter.CostBasis.Get()
		switch {
		case costBasis == nil && r.CostBasis != nil:
			return false
		case costBasis != nil && r.CostBasis == nil:
			return false
		case costBasis != nil && r.CostBasis != nil && !costBasis.Equal(*r.CostBasis):
			return false
		}
	}
	if filter.CreditPriority != nil && (r.CreditPriority == nil || *r.CreditPriority != *filter.CreditPriority) {
		return false
	}
	if filter.TransactionAuthorizationStatus != nil && (r.TransactionAuthorizationStatus == nil || *r.TransactionAuthorizationStatus != *filter.TransactionAuthorizationStatus) {
		return false
	}

	return true
}

// Normalize canonicalizes route values so semantically equivalent routes share
// the same stored literals and routing keys. It also sets Version based on the
// fields present in the route.
func (r Route) Normalize() (Route, error) {
	if err := r.Validate(); err != nil {
		return Route{}, err
	}

	normalized := r
	if normalized.ExchangeSourceCurrency != nil && *normalized.ExchangeSourceCurrency == "" {
		normalized.ExchangeSourceCurrency = nil
	}
	normalized.Features = SortedFeatures(r.Features)
	normalized.Version = selectRoutingKeyVersion(normalized)

	return normalized, nil
}

// Normalize canonicalizes route filter values before querying.
func (f RouteFilter) Normalize() (RouteFilter, error) {
	if f.Currency == "" && f.ExchangeSourceCurrency.IsAbsent() && f.TaxCode.IsAbsent() && f.Features.IsAbsent() && f.MatchFeature == "" && f.CostBasis.IsAbsent() && f.CreditPriority == nil && f.TransactionAuthorizationStatus == nil && f.TaxBehavior.IsAbsent() {
		return f, nil
	}
	if f.Features.IsPresent() && f.MatchFeature != "" {
		return RouteFilter{}, errors.New("features and match feature filters cannot be combined")
	}
	if f.MatchFeature != "" {
		if err := validateFeatures([]string{f.MatchFeature}); err != nil {
			return RouteFilter{}, fmt.Errorf("match feature: %w", err)
		}
	}

	exchangeSourceCurrency, _ := f.ExchangeSourceCurrency.Get()
	taxCode, _ := f.TaxCode.Get()
	taxBehavior, _ := f.TaxBehavior.Get()
	features, _ := f.Features.Get()
	costBasis, _ := f.CostBasis.Get()
	normalized, err := Route{
		Currency:                       f.Currency,
		ExchangeSourceCurrency:         exchangeSourceCurrency,
		TaxCode:                        taxCode,
		TaxBehavior:                    taxBehavior,
		Features:                       features,
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

	normalizedExchangeSourceCurrency := mo.None[*currencyx.Code]()
	if f.ExchangeSourceCurrency.IsPresent() {
		normalizedExchangeSourceCurrency = mo.Some(normalized.ExchangeSourceCurrency)
	}

	normalizedTaxCode := mo.None[*string]()
	if f.TaxCode.IsPresent() {
		normalizedTaxCode = mo.Some(normalized.TaxCode)
	}

	normalizedTaxBehavior := mo.None[*TaxBehavior]()
	if f.TaxBehavior.IsPresent() {
		normalizedTaxBehavior = mo.Some(normalized.TaxBehavior)
	}

	normalizedFeatures := mo.None[[]string]()
	if f.Features.IsPresent() {
		normalizedFeatures = mo.Some(normalized.Features)
	}

	return RouteFilter{
		Currency:                       normalized.Currency,
		ExchangeSourceCurrency:         normalizedExchangeSourceCurrency,
		TaxCode:                        normalizedTaxCode,
		TaxBehavior:                    normalizedTaxBehavior,
		Features:                       normalizedFeatures,
		MatchFeature:                   f.MatchFeature,
		CostBasis:                      normalizedCostBasis,
		CreditPriority:                 normalized.CreditPriority,
		TransactionAuthorizationStatus: normalized.TransactionAuthorizationStatus,
	}, nil
}

// ----------------------------------------------------------------------------
// Routing key generation
// ----------------------------------------------------------------------------

// routingVersionRequirement pairs a version with the route condition that requires it.
type routingVersionRequirement struct {
	version  RoutingKeyVersion
	requires func(Route) bool
}

// routingVersionRequirements lists versions above V1 with the conditions that trigger them.
// Ordered highest to lowest; selectRoutingKeyVersion returns the first match, V1 otherwise.
var routingVersionRequirements = []routingVersionRequirement{
	{version: RoutingKeyVersionV3, requires: func(r Route) bool { return r.ExchangeSourceCurrency != nil }},
	{version: RoutingKeyVersionV2, requires: func(r Route) bool { return r.TaxBehavior != nil }},
}

// selectRoutingKeyVersion returns the minimum routing key version required to
// encode all fields present in route. V1 is the baseline; higher versions are
// selected when the route uses fields that V1 does not include.
func selectRoutingKeyVersion(route Route) RoutingKeyVersion {
	for _, req := range routingVersionRequirements {
		if req.requires(route) {
			return req.version
		}
	}
	return RoutingKeyVersionV1
}

// BuildRoutingKey normalizes route and encodes it as a RoutingKey.
// The version is determined automatically from the route fields present.
func BuildRoutingKey(route Route) (RoutingKey, error) {
	normalizedRoute, err := route.Normalize()
	if err != nil {
		return RoutingKey{}, err
	}

	switch normalizedRoute.Version {
	case RoutingKeyVersionV1:
		return buildRoutingKeyV1Normalized(normalizedRoute)
	case RoutingKeyVersionV2:
		return buildRoutingKeyV2Normalized(normalizedRoute)
	case RoutingKeyVersionV3:
		return buildRoutingKeyV3Normalized(normalizedRoute)
	default:
		return RoutingKey{}, ErrRoutingKeyVersionUnsupported.WithAttrs(models.Attributes{
			"routing_key_version": normalizedRoute.Version,
		})
	}
}

// BuildRoutingKeyV1 encodes route as a V1 routing key.
// Returns an error if route.TaxBehavior is non-nil; use BuildRoutingKey to
// select the correct version automatically based on route fields.
func BuildRoutingKeyV1(route Route) (RoutingKey, error) {
	if route.TaxBehavior != nil {
		return RoutingKey{}, fmt.Errorf("TaxBehavior requires a V2 routing key; use BuildRoutingKey to select the version automatically")
	}
	normalizedRoute, err := route.Normalize()
	if err != nil {
		return RoutingKey{}, err
	}
	if normalizedRoute.ExchangeSourceCurrency != nil {
		return RoutingKey{}, fmt.Errorf("ExchangeSourceCurrency requires a V3 routing key; use BuildRoutingKey to select the version automatically")
	}
	return buildRoutingKeyV1Normalized(normalizedRoute)
}

// BuildRoutingKeyV2 encodes route as a V2 routing key.
func BuildRoutingKeyV2(route Route) (RoutingKey, error) {
	normalizedRoute, err := route.Normalize()
	if err != nil {
		return RoutingKey{}, err
	}
	if normalizedRoute.ExchangeSourceCurrency != nil {
		return RoutingKey{}, fmt.Errorf("ExchangeSourceCurrency requires a V3 routing key; use BuildRoutingKey to select the version automatically")
	}
	return buildRoutingKeyV2Normalized(normalizedRoute)
}

// BuildRoutingKeyV3 encodes route as a V3 routing key.
func BuildRoutingKeyV3(route Route) (RoutingKey, error) {
	normalizedRoute, err := route.Normalize()
	if err != nil {
		return RoutingKey{}, err
	}
	return buildRoutingKeyV3Normalized(normalizedRoute)
}

// buildRoutingKeyV1Normalized encodes an already-normalized route as a V1 key.
func buildRoutingKeyV1Normalized(route Route) (RoutingKey, error) {
	value := strings.Join([]string{
		"currency:" + string(route.Currency),
		"tax_code:" + optionalStringValue(route.TaxCode),
		"features:" + canonicalFeatures(route.Features),
		"cost_basis:" + optionalDecimalValue(route.CostBasis),
		"credit_priority:" + optionalIntValue(route.CreditPriority),
		"transaction_authorization_status:" + string(lo.FromPtrOr(route.TransactionAuthorizationStatus, "null")),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV1, value)
}

// buildRoutingKeyV2Normalized encodes an already-normalized route as a V2 key.
func buildRoutingKeyV2Normalized(route Route) (RoutingKey, error) {
	value := strings.Join([]string{
		"currency:" + string(route.Currency),
		"tax_code:" + optionalStringValue(route.TaxCode),
		"tax_behavior:" + string(lo.FromPtrOr(route.TaxBehavior, "null")),
		"features:" + canonicalFeatures(route.Features),
		"cost_basis:" + optionalDecimalValue(route.CostBasis),
		"credit_priority:" + optionalIntValue(route.CreditPriority),
		"transaction_authorization_status:" + string(lo.FromPtrOr(route.TransactionAuthorizationStatus, "null")),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV2, value)
}

func buildRoutingKeyV3Normalized(route Route) (RoutingKey, error) {
	value := strings.Join([]string{
		"currency:" + string(route.Currency),
		"exchange_source_currency:" + string(lo.FromPtrOr(route.ExchangeSourceCurrency, currencyx.Code("null"))),
		"tax_code:" + optionalStringValue(route.TaxCode),
		"tax_behavior:" + string(lo.FromPtrOr(route.TaxBehavior, "null")),
		"features:" + canonicalFeatures(route.Features),
		"cost_basis:" + optionalDecimalValue(route.CostBasis),
		"credit_priority:" + optionalIntValue(route.CreditPriority),
		"transaction_authorization_status:" + string(lo.FromPtrOr(route.TransactionAuthorizationStatus, "null")),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV3, value)
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

func ValidateExchangeSourceCurrency(currency currencyx.Code, exchangeSourceCurrency *currencyx.Code) error {
	if exchangeSourceCurrency == nil || *exchangeSourceCurrency == "" {
		return nil
	}

	if err := ValidateCurrency(*exchangeSourceCurrency); err != nil || goblcurrency.Get(goblcurrency.Code(*exchangeSourceCurrency)) == nil {
		return ErrCurrencyInvalid.WithAttrs(models.Attributes{
			"exchange_source_currency": *exchangeSourceCurrency,
			"reason":                   "exchange_source_currency_must_be_fiat",
		})
	}

	if goblcurrency.Get(goblcurrency.Code(currency)) != nil {
		return ErrCurrencyInvalid.WithAttrs(models.Attributes{
			"currency":                 currency,
			"exchange_source_currency": *exchangeSourceCurrency,
			"reason":                   "fiat_exchange_source_currency_must_be_null",
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

// ValidateFeatures validates feature route keys.
func ValidateFeatures(features []string) error {
	return validateFeatures(features)
}

func validateFeatures(features []string) error {
	var errs []error
	seen := make(map[string]struct{}, len(features))

	for i, feature := range features {
		if feature == "" {
			errs = append(errs, fmt.Errorf("[%d]: feature key is required", i))
			continue
		}

		if _, ok := seen[feature]; ok {
			errs = append(errs, fmt.Errorf("[%d]: duplicate feature key", i))
			continue
		}

		seen[feature] = struct{}{}
	}

	return errors.Join(errs...)
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
