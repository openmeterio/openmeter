package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateAPIInvoiceDeleteSupported is a temporary HTTP-level guard until
// usage-based invoice-scope deletion is implemented. It blocks the public API
// before standard DeleteInvoice or gathering DeleteGatheringInvoice can run
// side-effectful line-engine cleanup on other charge-backed lines in the same
// invoice.
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
		standardInvoice, err := invoice.AsStandardInvoice()
		if err != nil {
			return err
		}
		if standardInvoice.DeletedAt != nil {
			return nil
		}
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

		// Usage-based charge deletion at invoice scope is not implemented yet.
		// Keep this temporary HTTP-only guard ahead of both standard and
		// gathering invoice deletion so mixed invoices cannot run flat-fee
		// cleanup before a usage-based line rejects.
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
