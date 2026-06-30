package meta

import (
	"errors"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFieldsRejectsFeatureKeyChange(t *testing.T) {
	err := ValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFields(billing.InvoiceLineOverride{
		ChangesToApply: billing.ExistingLineOverride{
			FeatureKey: mo.Some("new-feature"),
		},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, billing.ErrInvoiceLineFeatureKeyEditNotSupported))
}

func TestValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFieldsRejectsTaxConfigChange(t *testing.T) {
	err := ValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFields(billing.InvoiceLineOverride{
		ChangesToApply: billing.ExistingLineOverride{
			TaxConfig: mo.Some(&billing.TaxConfig{}),
		},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, billing.ErrInvoiceLineTaxConfigEditNotSupported))
}
