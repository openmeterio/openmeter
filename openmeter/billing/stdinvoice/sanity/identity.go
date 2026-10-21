package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateIdentities requires IDs and matching invoice ownership on active lines
// and details. Line, detail and discount IDs must each be unique within their kind:
// apps use them as map keys and update targets, so duplicates can drop entries or
// update the wrong one. Child references need only be unique within a parent.
// Provider IDs are optional because the app may not have created the invoice yet.
func validateIdentities(invoice billing.StandardInvoice) error {
	var errs []error
	if invoice.ID == "" {
		errs = append(errs, billing.ValidationWithFieldPrefix("id", ErrMissingIdentity))
	}

	if invoice.Namespace == "" {
		errs = append(errs, billing.ValidationWithFieldPrefix("namespace", ErrMissingIdentity))
	}

	lineIDs := identitySet{}
	detailIDs := identitySet{}
	discountIDs := identitySet{}
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		lineErrs := []error{billing.ValidationWithFieldPrefix("id", lineIDs.check(line.ID))}
		ownership := invoiceOwnership{Invoice: invoice, InvoiceID: line.InvoiceID, Namespace: line.Namespace}
		lineErrs = append(lineErrs, ownership.Validate())
		childRefs := identitySet{}
		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			ownership = invoiceOwnership{Invoice: invoice, InvoiceID: detail.InvoiceID, Namespace: detail.Namespace}
			detailErrs := []error{billing.ValidationWithFieldPrefix("id", detailIDs.check(detail.ID)), billing.ValidationWithFieldPrefix("childUniqueReferenceID", childRefs.check(detail.ChildUniqueReferenceID)), ownership.Validate()}
			for discountIdx, discount := range detail.AmountDiscounts {
				if discount.DeletedAt != nil {
					continue
				}

				detailErrs = append(detailErrs, billing.ValidationWithFieldPrefix(fmt.Sprintf("amountDiscounts/%d/id", discountIdx), discountIDs.check(discount.ID)))
			}

			lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix(fmt.Sprintf("detailedLines/%d", detailIdx), billing.ValidationWithAttributes(models.Annotations{"detailed_line_id": detail.ID}, errors.Join(detailErrs...))))
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d", lineIdx), billing.ValidationWithAttributes(models.Annotations{"line_id": line.ID}, errors.Join(lineErrs...))))
	}

	return billing.ValidationWithAttributes(models.Annotations{"invoice_id": invoice.ID}, errors.Join(errs...))
}

type identitySet map[string]struct{}

// check reports missing and duplicate IDs separately. Callers use separate sets
// so, for example, a line and a discount may have the same ID.
func (seen identitySet) check(id string) error {
	if id == "" {
		return ErrMissingIdentity
	}

	if _, ok := seen[id]; ok {
		return billing.ValidationWithAttributes(models.Annotations{"actual": id}, ErrDuplicateIdentity)
	}

	seen[id] = struct{}{}

	return nil
}

type invoiceOwnership struct {
	Invoice   billing.StandardInvoice
	InvoiceID string
	Namespace string
}

func (o invoiceOwnership) Validate() error {
	var errs []error
	for _, field := range []struct{ name, expected, actual string }{{"invoiceID", o.Invoice.ID, o.InvoiceID}, {"namespace", o.Invoice.Namespace, o.Namespace}} {
		if field.expected != field.actual {
			errs = append(errs, billing.ValidationWithFieldPrefix(field.name, billing.ValidationWithAttributes(models.Annotations{"expected": field.expected, "actual": field.actual}, ErrOwnershipMismatch)))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
