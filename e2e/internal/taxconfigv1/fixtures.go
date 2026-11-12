// Package taxconfigv1 holds the helpers the v1 taxConfig characterization tests in e2e and
// e2e/creditsdisabled share: tax code and customer fixtures, invoice line builders, and the
// assertions that pin the rendered tax configs. A change here changes what both suites pin.
package taxconfigv1

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// uniqueKey returns a fixture key suffix that survives re-runs against a shared
// database without collision. Format: <prefix>_<millis>_<rand>.
func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), rand.IntN(10_000))
}

// RandomStripeTaxCode returns a Stripe code that is very unlikely to be mapped already, since
// all e2e tests share one namespace and a mapped Stripe code resolves to the existing tax code.
func RandomStripeTaxCode() string {
	return fmt.Sprintf("txcd_%08d", rand.IntN(100_000_000))
}

// StripeAppMapping returns the Stripe tax code of the tax code's Stripe app mapping,
// or "" when it has none.
func StripeAppMapping(taxCode v3sdk.TaxCode) string {
	mapping, _ := lo.Find(taxCode.AppMappings, func(m v3sdk.TaxCodeAppMapping) bool {
		return m.AppType == v3sdk.AppTypeStripe
	})

	return mapping.TaxCode
}

// CreateCustomer creates a customer for the tax config scenarios, keyed by a unique
// keyPrefix-based fixture key.
func CreateCustomer(t *testing.T, client *api.ClientWithResponses, keyPrefix string) api.Customer {
	t.Helper()

	resp, err := client.CreateCustomerWithResponse(t.Context(), api.CustomerCreate{
		Name:     "v1 taxConfig " + keyPrefix,
		Key:      lo.ToPtr(uniqueKey(keyPrefix)),
		Currency: lo.ToPtr(api.CurrencyCode("USD")),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode(), "create customer: %s", string(resp.Body))
	require.NotNil(t, resp.JSON201)

	return *resp.JSON201
}
