package billingservice

import (
	"context"
	"time"

	"github.com/invopop/validation"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

var _ billing.InvoiceService = (*Service)(nil)

func (s *Service) GetPendingInvoiceItems(ctx context.Context, customerID customerentity.CustomerID) ([]billing.InvoiceWithValidation, error) {
	customerEntity, err := s.customerService.GetCustomer(ctx, customerentity.GetCustomerInput(customerID))
	if err != nil {
		if err, ok := lo.ErrorsAs[customerentity.NotFoundError](err); ok {
			return nil, billing.ValidationError{
				Err: err,
			}
		}

		return nil, err
	}

	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) ([]billing.InvoiceWithValidation, error) {
		validationErrors := []error{}

		billingProfile, err := s.getProfileWithCustomerOverride(ctx, adapter, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  customerEntity.Namespace,
			CustomerID: customerEntity.ID,
		})
		if err != nil {
			// If the customer has no billing profile, we can't create an invoice, but for pending items we can
			// report the error and the pending items, so that the caller can decide what to do

			if err, ok := lo.ErrorsAs[validation.Error](err); !ok {
				return nil, err
			}

			validationErrors = append(validationErrors, err)
		}

		pendingItems, err := s.adapter.GetPendingInvoiceItems(ctx, customerID)
		if err != nil {
			// If we cannot get the pending items, we can bail here, as the caller can't do anything
			return nil, err
		}

		// We don't support multi-currency invoices (as that would require up-to-date exchange rates etc.), so
		// let's split the pending invoice items into per currency invoices

		byCurrency := splitInvoicesByCurrency(pendingItems)

		res := make([]billing.InvoiceWithValidation, 0, len(byCurrency))

		for currency, items := range byCurrency {
			res = append(res, billing.InvoiceWithValidation{
				Invoice: &billing.Invoice{
					Namespace: customerEntity.Namespace,
					InvoiceNumber: billing.InvoiceNumber{
						Series: "INV",
						Code:   "DRAFT",
					},
					Status: billing.InvoiceStatusPendingCreation,
					Items:  items,
					Type:   billing.InvoiceTypeStandard,

					// TODO[OM-931]: Timezone

					// TODO: Period is not captured here, but it should be fine
					Currency:  currency,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),

					Profile:  billingProfile.Profile,
					Customer: billing.InvoiceCustomer(billingProfile.Customer),
				},
				ValidationErrors: validationErrors,
			},
			)
		}

		return res, nil
	})
}

func splitInvoicesByCurrency(items []billing.InvoiceItem) map[currencyx.Code][]billing.InvoiceItem {
	byCurrency := make(map[currencyx.Code][]billing.InvoiceItem)

	if len(items) == 0 {
		return byCurrency
	}

	// Optimization: pre-allocate the first currency, assuming that there will be not more than one currency
	byCurrency[items[0].Currency] = make([]billing.InvoiceItem, 0, len(items))

	for _, item := range items {
		byCurrency[item.Currency] = append(byCurrency[item.Currency], item)
	}

	return byCurrency
}
