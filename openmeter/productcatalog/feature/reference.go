package feature

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FeatureReference identifies a feature by ID, key, or both. Partial
// references are valid authoring inputs; persistence boundaries must require a
// complete identity after resolution.
type FeatureReference struct {
	ID  *string `json:"id,omitempty"`
	Key *string `json:"key,omitempty"`

	// resolved contains the runtime feature representation and is deliberately
	// omitted from serialization and identity comparisons.
	resolved *Feature
}

func (r FeatureReference) Validate() error {
	var errs []error

	if r.ID != nil && *r.ID == "" {
		errs = append(errs, errors.New("id cannot be empty"))
	}

	if r.Key != nil && *r.Key == "" {
		errs = append(errs, errors.New("key cannot be empty"))
	}

	if r.ID == nil && r.Key == nil {
		errs = append(errs, errors.New("id or key is required"))
	}

	if r.resolved != nil {
		if err := r.resolved.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("resolved feature: %w", err))
		} else {
			if r.ID != nil && *r.ID != r.resolved.ID {
				errs = append(errs, fmt.Errorf("id mismatch between reference and resolved feature [reference.id=%s resolved.id=%s]", *r.ID, r.resolved.ID))
			}

			if r.Key != nil && *r.Key != r.resolved.Key {
				errs = append(errs, fmt.Errorf("key mismatch between reference and resolved feature [reference.key=%s resolved.key=%s]", *r.Key, r.resolved.Key))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// IsResolved reports whether the full feature representation is available.
func (r FeatureReference) IsResolved() bool {
	return r.resolved != nil
}

func (r FeatureReference) Feature() (*Feature, bool) {
	return r.resolved, r.resolved != nil
}

// Equal compares reference identity and deliberately ignores the runtime-only
// resolved feature.
func (r FeatureReference) Equal(other FeatureReference) bool {
	return equalOptionalString(r.ID, other.ID) && equalOptionalString(r.Key, other.Key)
}

// Compatible reports whether the identifiers supplied by both references do
// not conflict. It is intended for comparing complete stored references with
// partial authoring overlays.
func (r FeatureReference) Compatible(other FeatureReference) bool {
	if r.ID != nil && other.ID != nil && *r.ID != *other.ID {
		return false
	}

	if r.Key != nil && other.Key != nil && *r.Key != *other.Key {
		return false
	}

	return true
}

func (r FeatureReference) Clone() FeatureReference {
	if r.ID != nil {
		r.ID = lo.ToPtr(*r.ID)
	}

	if r.Key != nil {
		r.Key = lo.ToPtr(*r.Key)
	}

	if r.resolved != nil {
		resolved := r.resolved.Clone()
		r.resolved = &resolved
	}

	return r
}

// WithFeature validates the supplied identity, fills any missing identifier,
// and attaches the resolved feature.
func (r FeatureReference) WithFeature(feature *Feature) (FeatureReference, error) {
	if feature == nil {
		return FeatureReference{}, errors.New("feature is required")
	}

	if err := r.Validate(); err != nil {
		return FeatureReference{}, fmt.Errorf("invalid feature reference: %w", err)
	}

	if err := feature.Validate(); err != nil {
		return FeatureReference{}, fmt.Errorf("invalid resolved feature: %w", err)
	}

	if r.ID != nil && *r.ID != feature.ID {
		return FeatureReference{}, fmt.Errorf("id mismatch between reference and feature [reference.id=%s resolved.id=%s]", *r.ID, feature.ID)
	}

	if r.Key != nil && *r.Key != feature.Key {
		return FeatureReference{}, fmt.Errorf("key mismatch between reference and feature [reference.key=%s resolved.key=%s]", *r.Key, feature.Key)
	}

	r.ID = lo.ToPtr(feature.ID)
	r.Key = lo.ToPtr(feature.Key)
	r.resolved = feature

	if err := r.Validate(); err != nil {
		return FeatureReference{}, err
	}

	return r, nil
}

func (f Feature) Reference() FeatureReference {
	return FeatureReference{
		ID:       lo.ToPtr(f.ID),
		Key:      lo.ToPtr(f.Key),
		resolved: &f,
	}
}

// Clone creates a deep copy of the mutable values carried by a feature.
func (f Feature) Clone() Feature {
	if f.Description != nil {
		f.Description = lo.ToPtr(*f.Description)
	}

	if f.MeterID != nil {
		f.MeterID = lo.ToPtr(*f.MeterID)
	}

	if f.MeterSlug != nil {
		f.MeterSlug = lo.ToPtr(*f.MeterSlug)
	}

	if f.MeterGroupByFilters != nil {
		filters := make(MeterGroupByFilters, len(f.MeterGroupByFilters))
		for key, value := range f.MeterGroupByFilters {
			filters[key] = cloneFilterString(value)
		}
		f.MeterGroupByFilters = filters
	}

	if f.UnitCost != nil {
		unitCost := *f.UnitCost
		if unitCost.Manual != nil {
			unitCost.Manual = lo.ToPtr(*unitCost.Manual)
		}
		if unitCost.LLM != nil {
			unitCost.LLM = lo.ToPtr(*unitCost.LLM)
		}
		f.UnitCost = &unitCost
	}

	f.Metadata = maps.Clone(f.Metadata)

	if f.ArchivedAt != nil {
		f.ArchivedAt = lo.ToPtr(*f.ArchivedAt)
	}

	return f
}

func equalOptionalString(left, right *string) bool {
	if (left == nil) != (right == nil) {
		return false
	}

	return left == nil || *left == *right
}

func cloneFilterString(value filter.FilterString) filter.FilterString {
	clone := value

	clone.Eq = cloneString(value.Eq)
	clone.Ne = cloneString(value.Ne)
	clone.Exists = cloneBool(value.Exists)
	clone.Like = cloneString(value.Like)
	clone.Nlike = cloneString(value.Nlike)
	clone.Ilike = cloneString(value.Ilike)
	clone.Nilike = cloneString(value.Nilike)
	clone.Contains = cloneString(value.Contains)
	clone.Ncontains = cloneString(value.Ncontains)
	clone.Gt = cloneString(value.Gt)
	clone.Gte = cloneString(value.Gte)
	clone.Lt = cloneString(value.Lt)
	clone.Lte = cloneString(value.Lte)

	if value.In != nil {
		clone.In = lo.ToPtr(slices.Clone(*value.In))
	}
	if value.Nin != nil {
		clone.Nin = lo.ToPtr(slices.Clone(*value.Nin))
	}
	if value.And != nil {
		and := make([]filter.FilterString, len(*value.And))
		for idx, nested := range *value.And {
			and[idx] = cloneFilterString(nested)
		}
		clone.And = &and
	}
	if value.Or != nil {
		or := make([]filter.FilterString, len(*value.Or))
		for idx, nested := range *value.Or {
			or[idx] = cloneFilterString(nested)
		}
		clone.Or = &or
	}

	return clone
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}

	return lo.ToPtr(*value)
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}

	return lo.ToPtr(*value)
}
