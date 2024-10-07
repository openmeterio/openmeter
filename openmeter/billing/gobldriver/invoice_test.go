package gobldriver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidationErrors(t *testing.T) {
	now := time.Now()
	billingPeriodStart := now.AddDate(0, -1, 0)
	billingPeriodEnd := now.AddDate(0, 1, 0)

	invoice := billing.InvoiceWithValidation{
		Invoice: &billing.Invoice{
			Currency: currencyx.Code(currency.USD),
			Items: []billing.InvoiceItem{
				{
					Name:        "item in usd",
					Quantity:    lo.ToPtr(alpacadecimal.NewFromFloat(1)),
					UnitPrice:   alpacadecimal.NewFromFloat(100),
					Currency:    "USD",
					PeriodStart: billingPeriodStart,
					PeriodEnd:   billingPeriodEnd,
					CreatedAt:   billingPeriodStart,
				},
				{
					Name:        "item in huf",
					Quantity:    lo.ToPtr(alpacadecimal.NewFromFloat(2)),
					UnitPrice:   alpacadecimal.NewFromFloat(200),
					Currency:    "HUF",
					PeriodStart: billingPeriodStart,
					PeriodEnd:   billingPeriodEnd,
					CreatedAt:   billingPeriodStart,
				},
				{
					// This tests the line item conversion validation
					Name:        "item with too big quantity huf",
					Quantity:    lo.ToPtr(lo.Must(alpacadecimal.NewFromString("19223372036854775807"))),
					UnitPrice:   alpacadecimal.NewFromFloat(200),
					Currency:    "HUF",
					PeriodStart: billingPeriodStart,
					PeriodEnd:   billingPeriodEnd,
					CreatedAt:   billingPeriodStart,
				},
			},
		},
		ValidationErrors: []error{
			errors.New("generic error"),
			billing.NotFoundError{
				Err: billing.ErrDefaultProfileNotFound,
			},
		},
	}

	d := Driver{
		logger: slog.Default(),
	}

	inv, err := d.Generate(context.Background(), invoice)
	require.NoError(t, err)
	require.NotNil(t, inv)
	require.Len(t, inv.Complements, 1)

	validationError, err := LookupValidationErrors(inv)
	require.NoError(t, err)
	require.NotNil(t, validationError)

	expectedOutput := ValidationErrorsComplement{
		Fields: map[string][]ValidationErrorJSON{
			"customer.billingAddress": {
				{Code: "customer_billing_address_not_found", Message: "missing customer billing address"},
			},
			"lines.1": {
				{Message: "no exchange rate found from 'HUF' to 'USD'"},
			},
			"lines.2.quantity": {
				{Code: "number_conversion", Message: "error converting quantity: invalid major number '19223372036854775807', strconv.ParseInt: parsing \"19223372036854775807\": value out of range"},
			},
			"supplier.name": {
				{Code: "validation_required", Message: "cannot be blank"},
			},
			"type": {
				{Code: "validation_required", Message: "cannot be blank"},
			},
		},
		Global: []ValidationErrorJSON{
			{Message: "generic error"},
			{Code: "default_profile_not_found", Message: "default profile not found"},
			{Code: "missing_payment_method", Message: "missing payment method"},
		},
	}

	require.Equal(t, expectedOutput, validationError)

	// Let's validate marshaling unmarshaling
	t.Run("json tags", func(t *testing.T) {
		marshaledVE, err := json.Marshal(validationError)
		require.NoError(t, err)
		require.NotEmpty(t, marshaledVE)

		parsedValidationError := ValidationErrorsComplement{}
		require.NoError(t, json.Unmarshal(marshaledVE, &parsedValidationError))

		require.Equal(t, validationError, parsedValidationError)
	})
}
