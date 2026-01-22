package appcustominvoicing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SyncDraftInvoiceInput struct {
	InvoiceID            billing.InvoiceID
	UpsertInvoiceResults *billing.UpsertStandardInvoiceResult
}

func (i *SyncDraftInvoiceInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.UpsertInvoiceResults == nil {
		errs = append(errs, fmt.Errorf("upsert invoice results are required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type SyncIssuingInvoiceInput struct {
	InvoiceID             billing.InvoiceID
	FinalizeInvoiceResult *billing.FinalizeStandardInvoiceResult
}

func (i *SyncIssuingInvoiceInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.FinalizeInvoiceResult == nil {
		errs = append(errs, fmt.Errorf("finalize invoice result is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type HandlePaymentTriggerInput struct {
	InvoiceID billing.InvoiceID
	Trigger   billing.InvoiceTrigger
}

func (i *HandlePaymentTriggerInput) Validate() error {
	var errs []error

	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Trigger == "" {
		errs = append(errs, fmt.Errorf("trigger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
