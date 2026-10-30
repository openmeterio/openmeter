package meta_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestIntentRequiresTaxCodeID(t *testing.T) {
	intent := meta.Intent{
		ManagedBy:  billing.SystemManagedLine,
		CustomerID: "01JBP3SGZ20Y7VRVC351TDFXYZ",
		Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: "01AN4Z07BY79KA1307SR9X4MV3",
		},
	}

	require.NoError(t, intent.Validate())

	// TaxCodeConfig accepts a behavior-only config, but a charge always needs a Tax Code.
	intent.TaxConfig = productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)}

	require.ErrorContains(t, intent.Validate(), "tax config: tax code id is required")
}
