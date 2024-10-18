package gobldriver

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"cloud.google.com/go/civil"
	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/validation"
	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/gobl"
)

type Driver struct {
	logger *slog.Logger
}

type DriverConfig struct {
	Logger *slog.Logger
}

func (c DriverConfig) Validate() error {
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	return nil
}

func NewDriver(config DriverConfig) (*Driver, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("error validating driver config: %w", err)
	}

	return &Driver{
		logger: config.Logger,
	}, nil
}

type invoiceWithValidation struct {
	Invoice          *bill.Invoice
	ValidationErrors []error
}

// Generate converts a billing.Invoice to a gobldriver.Invoice
// if error is set, then a non-recordable error occurred
// if the invoice has validation issues then the Invoice's ValidationError will be set
// and the Invoice's Complements will contain the validation notes
func (d *Driver) Generate(ctx context.Context, invoice billingentity.InvoiceWithValidation) (*bill.Invoice, error) {
	inv := d.invoiceToGOBL(invoice)

	if err := inv.Invoice.Validate(); err != nil {
		inv.ValidationErrors = append(inv.ValidationErrors, err)
	}

	if err := inv.Invoice.Calculate(); err != nil {
		inv.ValidationErrors = append(inv.ValidationErrors, err)
	}

	// TODO[OM-929]: Hook in all provider specific validations (see openmeter/provider/api.go)

	if len(inv.ValidationErrors) > 0 {
		complement, err := NewValidationErrorsComplement(inv.ValidationErrors)
		if err != nil {
			return nil, fmt.Errorf("error creating validation errors complement: %w", err)
		}

		object, err := schema.NewObject(complement)
		if err != nil {
			return nil, fmt.Errorf("error creating schema object: %w", err)
		}

		inv.Invoice.Complements = append(inv.Invoice.Complements, object)
	}

	return inv.Invoice, nil
}

// invoiceToGOBL converts a billing.Invoice to a gobl invoice. If error is set, then
// a fatal error has occurred, which should be reported to the caller instead of putting
// it into the validation errors.
func (d *Driver) invoiceToGOBL(input billingentity.InvoiceWithValidation) invoiceWithValidation {
	validationErrors := slices.Clone(input.ValidationErrors)

	inv := input.Invoice

	loc, err := inv.Timezone.LoadLocation()
	if err != nil {
		validationErrors = append(validationErrors, NewWithMessage(
			ErrLoadingTimezoneLocation,
			fmt.Sprintf("error loading timezone location[%s]: %v", inv.Timezone, err),
		))

		// fallback to UTC
		loc = time.UTC
	}

	customer, vErr := d.invoiceCustomerToParty(inv.Customer)
	if len(vErr) > 0 {
		validationErrors = append(validationErrors, validation.Errors{
			"customer": vErr,
		})
	}

	invoice := &bill.Invoice{
		Type:   inv.Type.CBCKey(),
		Series: cbc.Code(inv.InvoiceNumber.Series),
		Code:   cbc.Code(inv.InvoiceNumber.Code),
		IssueDate: cal.Date{
			Date: civil.DateOf(lo.FromPtrOr(inv.IssuedAt, inv.CreatedAt).In(loc)),
		},
		Currency: currency.Code(inv.Currency),
		Supplier: d.invoiceSupplierContactToParty(inv.Profile.Supplier),
		Customer: customer,
		Totals:   &bill.Totals{},
		Meta:     gobl.MetadataToGOBLMeta(inv.Metadata),
	}

	// NOTE: We need to add this to the Meta as gobl only supports UUID IDs
	invoice.Meta[InvoiceIDKey] = inv.ID

	switch inv.Profile.WorkflowConfig.Payment.CollectionMethod {
	case billingentity.CollectionMethodChargeAutomatically:
		invoice.Payment = &bill.Payment{
			Terms: &pay.Terms{
				Key: pay.TermKeyInstant,
			},
		}

	case billingentity.CollectionMethodSendInvoice:
		invoice.Payment = &bill.Payment{
			Terms: &pay.Terms{
				Key: pay.TermKeyDueDate,
				DueDates: []*pay.DueDate{
					{
						Date: &cal.Date{
							Date: civil.DateOf(inv.DueDate.In(loc)),
						},
					},
				},
			},
		}
	default:
		// Most probably caused by missing billing profile/override
		validationErrors = append(validationErrors, ErrMissingPaymentMethod)
	}

	var lineValidationErrors validation.Errors

	for idx, item := range inv.Items {
		line, err := d.invoiceItemToLine(item, loc)
		if err != nil {
			lineValidationErrors = upsertErrors(lineValidationErrors)
			lineValidationErrors[strconv.Itoa(idx)] = err
		}

		// Set the index of the line, as validation makes this required (indexing starts from 1
		// as the validator considers 0 as empty value for index)
		line.Index = idx + 1

		invoice.Lines = append(invoice.Lines, line)
	}

	if len(lineValidationErrors) > 0 {
		// Let's prepend the lines to the validation errors so that the json structure is correct
		validationErrors = append(validationErrors, validation.Errors{
			"lines": lineValidationErrors,
		})
	}

	// TODO: Series will most probably end up in Complements

	return invoiceWithValidation{
		Invoice:          invoice,
		ValidationErrors: validationErrors,
	}
}

func (d *Driver) invoiceItemToLine(item billingentity.InvoiceItem, loc *time.Location) (*bill.Line, validation.Errors) {
	var validationErrs validation.Errors

	// TODO: for usage based items we need to add a different logic
	quantity, err := gobl.DecimalToAmount(lo.FromPtrOr(item.Quantity, alpacadecimal.Zero))
	if err != nil {
		validationErrs = upsertErrors(validationErrs)

		validationErrs["quantity"] = NewWithMessage(
			ErrNumberConversion,
			fmt.Sprintf("error converting quantity: %v", err))
	}

	unitPrice, err := gobl.DecimalToAmount(item.UnitPrice)
	if err != nil {
		validationErrs = upsertErrors(validationErrs)

		validationErrs["item"] = validation.Errors{
			"unitPrice": NewWithMessage(
				ErrNumberConversion,
				fmt.Sprintf("error converting unit price: %v", err)),
		}
	}

	return &bill.Line{
		Quantity: quantity,
		Item: &org.Item{
			// TODO: more fields from product catalog
			Name:     item.Name,
			Price:    unitPrice,
			Currency: currency.Code(item.Currency),
		},
		// NOTE: We need to add these as Notes as only the Items have Meta which is more on the
		// product catalog side, so it would be missleading to add the ID there
		Notes: []*cbc.Note{
			d.invoiceItemLifecycleNote(item, loc), // Billing period etc.
			d.invoiceItemEntityNote(item),
		},
	}, validationErrs
}

func (d *Driver) invoiceItemLifecycleNote(item billingentity.InvoiceItem, loc *time.Location) *cbc.Note {
	note := &cbc.Note{
		Key:  cbc.NoteKeyReason,
		Code: InvoiceItemCodeLifecycle,
		Src:  InvoiceItemNoteSourceOpenmeter,
		Meta: cbc.Meta{
			InvoiceItemBillingPeriodEnd:   item.PeriodEnd.In(loc).Format(time.RFC3339),
			InvoiceItemBillingPeriodStart: item.PeriodStart.In(loc).Format(time.RFC3339),
			InvoiceItemInvoiceAt:          item.InvoiceAt.In(loc).Format(time.RFC3339),
			InvoiceItemCreated:            item.CreatedAt.In(loc).Format(time.RFC3339),
			InvoiceItemUpdated:            item.UpdatedAt.In(loc).Format(time.RFC3339),
		},
	}

	// RFC1123 is used for human readability (as it has human readable timezone)
	// The Meta fields contain the RFC3339 format for machine readability
	note.Text = fmt.Sprintf("Billing period: %s - %s",
		item.PeriodStart.In(loc).Format(time.RFC1123),
		item.PeriodEnd.In(loc).Format(time.RFC1123))

	return note
}

func (d *Driver) invoiceItemEntityNote(item billingentity.InvoiceItem) *cbc.Note {
	note := &cbc.Note{
		Key:  cbc.NoteKeyGeneral,
		Code: InvoiceItemCodeEntity,
		Src:  InvoiceItemNoteSourceOpenmeter,
		Text: fmt.Sprintf("reference: %s", item.ID),
		Meta: cbc.Meta{
			InvoiceItemEntityID: item.ID,
		},
	}

	return note
}

func (d *Driver) invoiceCustomerToParty(i billingentity.InvoiceCustomer) (*org.Party, validation.Errors) {
	if i.BillingAddress == nil {
		return nil, validation.Errors{
			"billingAddress": ErrMissingCustomerBillingAddress,
		}
	}
	party := &org.Party{
		Name: i.Name,
		Addresses: []*org.Address{
			{
				Country:     l10n.ISOCountryCode(lo.FromPtrOr(i.BillingAddress.Country, "")),
				Street:      lo.FromPtrOr(i.BillingAddress.Line1, ""),
				StreetExtra: lo.FromPtrOr(i.BillingAddress.Line2, ""),
				Region:      lo.FromPtrOr(i.BillingAddress.State, ""),
				Locality:    lo.FromPtrOr(i.BillingAddress.City, ""),

				Code: lo.FromPtrOr(i.BillingAddress.PostalCode, ""),
			},
		},
	}

	if i.BillingAddress.PhoneNumber != nil {
		party.Telephones = append(party.Telephones, &org.Telephone{
			Number: *i.BillingAddress.PhoneNumber,
		})
	}

	if i.PrimaryEmail != nil {
		party.Emails = append(party.Emails, &org.Email{
			Address: *i.PrimaryEmail,
		})
	}
	return party, nil
}

func (d *Driver) invoiceSupplierContactToParty(c billingentity.SupplierContact) *org.Party {
	party := &org.Party{
		Name: c.Name,
		Addresses: []*org.Address{
			{
				Country:     l10n.ISOCountryCode(lo.FromPtrOr(c.Address.Country, "")),
				Street:      lo.FromPtrOr(c.Address.Line1, ""),
				StreetExtra: lo.FromPtrOr(c.Address.Line2, ""),
				Region:      lo.FromPtrOr(c.Address.State, ""),
				Locality:    lo.FromPtrOr(c.Address.City, ""),

				Code: lo.FromPtrOr(c.Address.PostalCode, ""),
			},
		},
		// TODO[OM-931]: we might want to add an option tax ID too
		TaxID: &tax.Identity{
			Country: l10n.TaxCountryCode(lo.FromPtrOr(c.Address.Country, "")),
		},
	}

	// TODO[OM-932]: add email, phone, etc.
	return party
}
