package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateAPIInvoiceDeleteSupported is a temporary HTTP-level guard for
// gathering invoice deletion until usage-based gathering-line deletion is
// implemented.
func ValidateAPIInvoiceDeleteSupported(invoice Invoice) error {
	switch invoice.Type() {
	case InvoiceTypeGathering:
		gatheringInvoice, err := invoice.AsGatheringInvoice()
		if err != nil {
			return err
		}
		if gatheringInvoice.DeletedAt != nil {
			return nil
		}
	case InvoiceTypeStandard:
		return nil
	default:
		return models.NewNillableGenericValidationError(fmt.Errorf("invalid invoice type: %s", invoice.Type()))
	}

	genericInvoice, err := invoice.AsGenericInvoice()
	if err != nil {
		return err
	}

	return ValidateAPIGenericInvoiceDeleteSupported(genericInvoice)
}

func ValidateAPIGenericInvoiceDeleteSupported(invoice GenericInvoice) error {
	for _, line := range invoice.GetGenericLines().OrEmpty() {
		if line == nil || line.GetDeletedAt() != nil {
			continue
		}

		// Usage-based gathering-line deletion is not implemented yet. Keep this
		// temporary HTTP-only guard ahead of gathering invoice deletion so mixed
		// gathering invoices cannot run flat-fee cleanup before a usage-based
		// line rejects.
		if line.GetLineEngineType() == LineEngineTypeChargeUsageBased {
			return ValidationError{
				Err: ValidationWithComponent(
					LineEngineValidationComponent(LineEngineTypeChargeUsageBased),
					ErrCannotUpdateChargeManagedLine,
				),
			}
		}
	}

	return nil
}
