package ledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type CreditFiltersVersion int

const (
	CreditFiltersVersion1 CreditFiltersVersion = 1
	CreditFiltersVersion2 CreditFiltersVersion = 2
)

// CreditFilters combines dimensions with AND and entries within a dimension with OR.
// An empty dimension imposes no restriction. A restricted dimension does not
// match a route whose corresponding dimension is absent.
type CreditFilters struct {
	Version  CreditFiltersVersion `json:"schema_version"`
	Features []string             `json:"features,omitempty"`
	Plans    []PlanFilter         `json:"plans,omitempty"`
}

type FeatureFilters []string

type PlanFilter struct {
	Key     string         `json:"key"`
	Version *VersionFilter `json:"version,omitempty"`
}

// VersionFilter uses one integer comparison. Omission matches every version,
// including versions published after the grant was created.
type VersionFilter struct {
	Eq  *int  `json:"eq,omitempty"`
	In  []int `json:"in,omitempty"`
	Gte *int  `json:"gte,omitempty"`
	Lte *int  `json:"lte,omitempty"`
}

func (f FeatureFilters) Validate() error {
	var errs []error

	for i, key := range f {
		if key == "" {
			errs = append(errs, fmt.Errorf("[%d]: feature key is required", i))
		}
	}

	if len(f.Normalize()) != len(f) {
		errs = append(errs, errors.New("duplicate feature key"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f FeatureFilters) Normalize() FeatureFilters {
	return FeatureFilters(slicesx.Normalize([]string(f)))
}

// ValidateAsFeatureFilter validates the singular customer-facing filter form.
// Credit routes may be restricted to multiple features, but a spendability
// query can only ask for one feature at a time.
func (f FeatureFilters) ValidateAsFeatureFilter() error {
	switch len(f) {
	case 0:
		return errors.New("features are required when feature filter is restricted")
	case 1:
	default:
		return errors.New("feature filter supports exactly one feature")
	}

	if err := f.Validate(); err != nil {
		return fmt.Errorf("features: %w", err)
	}

	return nil
}

func (f CreditFilters) Validate() error {
	var errs []error

	switch f.effectiveVersion() {
	case CreditFiltersVersion1:
		if len(f.Plans) > 0 {
			errs = append(errs, fmt.Errorf("credit filters schema version %d cannot represent plans", f.Version))
		}
	case CreditFiltersVersion2:
	default:
		errs = append(errs, fmt.Errorf("unsupported credit filters schema version: %d", f.Version))
	}

	if err := FeatureFilters(f.Features).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("features: %w", err))
	}

	for i, plan := range f.Plans {
		if plan.Key == "" {
			errs = append(errs, fmt.Errorf("plans[%d]: key is required", i))
		}

		if plan.Version != nil {
			if err := plan.Version.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("plans[%d].version: %w", i, err))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f VersionFilter) Validate() error {
	var errs []error
	operators := 0

	for _, value := range []*int{f.Eq, f.Gte, f.Lte} {
		if value != nil {
			operators++

			if *value < 1 {
				errs = append(errs, errors.New("version must be positive"))
			}
		}
	}

	if f.In != nil {
		operators++

		if len(f.In) == 0 || len(f.In) > 100 {
			errs = append(errs, errors.New("in must contain between 1 and 100 versions"))
		}

		for _, value := range f.In {
			if value < 1 {
				errs = append(errs, errors.New("version must be positive"))
			}
		}
	}

	if operators != 1 {
		errs = append(errs, errors.New("exactly one version operator is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ValidateAsPlanFilter validates a balance or transaction selection: one plan,
// optionally pinned to a concrete version. Grant restrictions may use ranges.
func (f PlanFilter) ValidateAsPlanFilter() error {
	var errs []error

	if err := (CreditFilters{Plans: []PlanFilter{f}}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if f.Version != nil && f.Version.Eq == nil {
		errs = append(errs, errors.New("plan selection requires an exact version"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// effectiveVersion selects the oldest format that represents new filters.
// Explicit versions are retained so stored filters keep their encoding contract.
func (f CreditFilters) effectiveVersion() CreditFiltersVersion {
	if f.Version != 0 {
		return f.Version
	}

	if len(f.Plans) > 0 {
		return CreditFiltersVersion2
	}

	return CreditFiltersVersion1
}

func (f CreditFilters) Normalize() CreditFilters {
	out := CreditFilters{
		Version:  f.effectiveVersion(),
		Features: FeatureFilters(f.Features).Normalize(),
		Plans:    slices.Clone(f.Plans),
	}

	for i, plan := range out.Plans {
		if plan.Version == nil {
			continue
		}

		v := *plan.Version

		if v.Eq != nil {
			v.Eq = lo.ToPtr(*v.Eq)
		}

		if v.Gte != nil {
			v.Gte = lo.ToPtr(*v.Gte)
		}

		if v.Lte != nil {
			v.Lte = lo.ToPtr(*v.Lte)
		}

		v.In = slicesx.Normalize(v.In)
		if len(v.In) == 1 && v.Eq == nil && v.Gte == nil && v.Lte == nil {
			v.Eq, v.In = lo.ToPtr(v.In[0]), nil
		}

		out.Plans[i].Version = &v
	}

	slices.SortFunc(out.Plans, func(a, b PlanFilter) int {
		aJSON, _ := json.Marshal(a)
		bJSON, _ := json.Marshal(b)

		return bytes.Compare(aJSON, bJSON)
	})

	out.Plans = slices.CompactFunc(out.Plans, func(a, b PlanFilter) bool {
		aJSON, _ := json.Marshal(a)
		bJSON, _ := json.Marshal(b)

		return bytes.Equal(aJSON, bJSON)
	})

	return out
}

func (f CreditFilters) IsEmpty() bool {
	return len(f.Features) == 0 && len(f.Plans) == 0
}

func (f CreditFilters) Equal(other CreditFilters) bool {
	return f.String() == other.String()
}

// String returns a canonical identity for route pairing, independent of the
// retained storage version. JSON encoding itself preserves the selected version.
func (f CreditFilters) String() string {
	f.Version = 0
	encoded, _ := json.Marshal(f.Normalize())

	return string(encoded)
}

// Matches compares restrictions with a route's recorded dimensions. This is
// directional: unrestricted filters match an unattributed route, but a
// restricted filter does not. It is not exact bucket equality.
func (f CreditFilters) Matches(route Route) bool {
	values := route.Filters

	if len(f.Features) > 0 && !slices.ContainsFunc(values.Features, func(key string) bool {
		return slices.Contains(f.Features, key)
	}) {
		return false
	}

	if len(f.Plans) == 0 {
		return true
	}

	return slices.ContainsFunc(f.Plans, func(filter PlanFilter) bool {
		return slices.ContainsFunc(values.Plans, func(value PlanFilter) bool {
			return filter.Key == value.Key && versionsOverlap(filter.Version, value.Version)
		})
	})
}

func versionsOverlap(a, b *VersionFilter) bool {
	if a == nil || b == nil {
		return true
	}

	if a.Eq != nil {
		return b.matches(*a.Eq)
	}

	if b.Eq != nil {
		return a.matches(*b.Eq)
	}

	if a.In != nil {
		return slices.ContainsFunc(a.In, b.matches)
	}

	if b.In != nil {
		return slices.ContainsFunc(b.In, a.matches)
	}

	if a.Gte != nil && b.Lte != nil {
		return *a.Gte <= *b.Lte
	}

	if b.Gte != nil && a.Lte != nil {
		return *b.Gte <= *a.Lte
	}

	return true
}

func (f VersionFilter) matches(version int) bool {
	switch {
	case f.Eq != nil:
		return version == *f.Eq
	case f.In != nil:
		return slices.Contains(f.In, version)
	case f.Gte != nil:
		return version >= *f.Gte
	case f.Lte != nil:
		return version <= *f.Lte
	default:
		return false
	}
}

func (f CreditFilters) MarshalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}

	filters := f.Normalize()

	switch filters.Version {
	case CreditFiltersVersion1:
		return json.Marshal(creditFiltersV1{Version: filters.Version, Features: filters.Features})
	case CreditFiltersVersion2:
		type plain CreditFilters

		return json.Marshal(plain(filters))
	default:
		return nil, fmt.Errorf("unsupported credit filters schema version: %d", f.Version)
	}
}

func (f *CreditFilters) UnmarshalJSON(data []byte) error {
	var header struct {
		SchemaVersion CreditFiltersVersion `json:"schema_version"`
	}

	// Parse the envelope first; the selected reader validates all payload fields.
	// Unmarshal also rejects trailing JSON values before version dispatch.
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var filters CreditFilters

	switch header.SchemaVersion {
	case CreditFiltersVersion1:
		var value creditFiltersV1

		if err := decoder.Decode(&value); err != nil {
			return err
		}

		filters = CreditFilters{Version: value.Version, Features: value.Features}
	case CreditFiltersVersion2:
		type plain CreditFilters
		var value plain

		if err := decoder.Decode(&value); err != nil {
			return err
		}

		filters = CreditFilters(value)
	default:
		return fmt.Errorf("unsupported credit filters schema version: %d", header.SchemaVersion)
	}

	if err := filters.Validate(); err != nil {
		return err
	}

	*f = filters.Normalize()

	return nil
}

// creditFiltersV1 keeps the feature-only reader strict when the current format adds plans.
type creditFiltersV1 struct {
	Version  CreditFiltersVersion `json:"schema_version"`
	Features []string             `json:"features,omitempty"`
}
