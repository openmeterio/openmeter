package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// MergeTaxConfigs merges two line tax configs with overrides taking precedence:
// the override's Tax Code ID wins when non-empty, and the behavior coalesces
// override → base. A nil override yields a clone of base and vice versa.
func MergeTaxConfigs(base, overrides *productcatalog.TaxCodeConfig) *productcatalog.TaxCodeConfig {
	if base != nil && overrides != nil {
		out := overrides.Clone()

		if out.TaxCodeID == "" {
			out.TaxCodeID = base.TaxCodeID
		}

		if out.Behavior == nil {
			out.Behavior = base.Behavior
		}

		return lo.ToPtr(out)
	}

	if overrides != nil {
		c := overrides.Clone()
		return &c
	}

	if base != nil {
		c := base.Clone()
		return &c
	}

	return nil
}

// taxCodeReferenceIssue returns nil for a live Tax Code, ErrTaxCodeNotFound when the namespace
// has no such row, and ErrTaxCodeDeleted when it is soft-deleted. The message names the ID.
func taxCodeReferenceIssue(id string, resolved productcatalog.TaxCodesByID) error {
	issue := ErrTaxCodeNotFound

	switch tc := resolved[id]; {
	case tc == nil:
	case tc.IsDeleted():
		issue = ErrTaxCodeDeleted
	default:
		return nil
	}

	issue.Message = fmt.Sprintf("%s [tax_code.id=%s]", issue.Message, id)

	return issue
}

// ValidateTaxCodeID rejects a missing or deleted Tax Code. Use it only for IDs the caller sent
// in this request: an ID copied from stored data may point at a Tax Code deleted since.
func ValidateTaxCodeID(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace, id string) error {
	if id == "" {
		return nil
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, id)
	if err != nil {
		return fmt.Errorf("resolving tax code %s: %w", id, err)
	}

	return taxCodeReferenceIssue(id, taxCodes)
}

// RequestTaxCodeRef is a Tax Code ID the caller sent, with the field path for its issues.
type RequestTaxCodeRef struct {
	TaxCodeID string
	// FieldPath locates the reference, e.g. "lines/0/tax_config/tax_code".
	FieldPath string
}

// ValidateRequestTaxCodeIDsInput carries the Tax Code IDs the caller sent in one request.
type ValidateRequestTaxCodeIDsInput struct {
	Namespace string
	Resolver  productcatalog.TaxCodeResolver
	// Duplicate IDs are resolved in one batch, but each failing reference gets its own issue.
	References []RequestTaxCodeRef
}

func (i ValidateRequestTaxCodeIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Resolver == nil {
		errs = append(errs, errors.New("resolver is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ValidateRequestTaxCodeIDs rejects Tax Code references that point at a missing or soft-deleted
// Tax Code: missing ones with ErrTaxCodeNotFound, deleted ones with ErrTaxCodeDeleted, each under
// the reference's field path. Use it only for IDs the caller sent in this request; IDs copied
// from stored data may point at a Tax Code deleted since. Empty IDs are skipped.
func ValidateRequestTaxCodeIDs(ctx context.Context, input ValidateRequestTaxCodeIDsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	ids := make([]string, 0, len(input.References))
	seen := make(map[string]struct{}, len(input.References))
	for _, ref := range input.References {
		if ref.TaxCodeID == "" {
			continue
		}

		if _, ok := seen[ref.TaxCodeID]; ok {
			continue
		}

		seen[ref.TaxCodeID] = struct{}{}
		ids = append(ids, ref.TaxCodeID)
	}

	if len(ids) == 0 {
		return nil
	}

	resolved, err := input.Resolver.BatchResolve(ctx, input.Namespace, ids...)
	if err != nil {
		return fmt.Errorf("resolving request tax codes: %w", err)
	}

	var errs []error
	for _, ref := range input.References {
		if ref.TaxCodeID == "" {
			continue
		}

		if err := taxCodeReferenceIssue(ref.TaxCodeID, resolved); err != nil {
			errs = append(errs, ValidationWithFieldPrefix(ref.FieldPath, err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return ValidationError{Err: err}
	}

	return nil
}
