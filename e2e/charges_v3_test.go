package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// chargeCustomerIDsByChargeID maps each listed flat fee charge to the customer
// ID it references.
func chargeCustomerIDsByChargeID(t *testing.T, items []v3sdk.Charge) map[string]string {
	t.Helper()

	out := make(map[string]string, len(items))
	for _, item := range items {
		flatFee, err := item.AsChargeFlatFee()
		require.NoError(t, err)
		ref, err := flatFee.Customer.AsCustomerReference()
		require.NoError(t, err)
		out[flatFee.ID] = ref.ID
	}

	return out
}

// TestV3ListCharges verifies the namespace-wide charges list: it serves the
// same charge shape as the customer-scoped list, narrows by the customer
// filter, and resolves each row's own customer under the customer expand.
func TestV3ListCharges(t *testing.T) {
	c := newV3Client(t)
	prefix := uniqueKey("charges_list")

	servicePeriod := v3sdk.ClosedPeriod{
		From: time.Now().UTC().Truncate(time.Second),
		To:   time.Now().UTC().Truncate(time.Second).Add(30 * 24 * time.Hour),
	}

	// given:
	// - two USD customers, each owning one flat fee charge
	type owner struct {
		customer *v3sdk.Customer
		chargeID string
	}
	owners := make([]owner, 0, 2)
	for _, suffix := range []string{"first", "second"} {
		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      uniqueKey(prefix + "_" + suffix),
			Name:     "Charges List " + suffix,
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)

		createReq, err := v3sdk.CreateChargeRequestFromCreateChargeFlatFeeRequest(v3sdk.CreateChargeFlatFeeRequest{
			Name:           "Flat Fee " + prefix + " " + suffix,
			Type:           v3sdk.ChargeTypeFlatFee,
			Currency:       "USD",
			InvoiceAt:      servicePeriod.To,
			ServicePeriod:  servicePeriod,
			SettlementMode: v3sdk.SettlementModeCreditThenInvoice,
			PaymentTerm:    v3sdk.PricePaymentTermInArrears,
			ProrationConfiguration: v3sdk.RateCardProrationConfiguration{
				Mode: v3sdk.RateCardProrationModeNoProration,
			},
			AmountBeforeProration: v3sdk.CurrencyAmount{Amount: "10", Currency: "USD"},
		})
		require.NoError(t, err)

		created, err := c.Customers.Charges.Create(t.Context(), customer.ID, createReq)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, created)
		flatFee, err := created.AsChargeFlatFee()
		require.NoError(t, err)

		owners = append(owners, owner{customer: customer, chargeID: flatFee.ID})
	}
	first, second := owners[0], owners[1]

	t.Run("filter by customer_id eq returns only that customer's charges", func(t *testing.T) {
		list, err := c.Charges.List(t.Context(), v3sdk.ListChargesParams{
			Filter: &v3sdk.ListChargesFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &first.customer.ID},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		assert.Equal(t, 1, list.Meta.Page.Total)
		require.Len(t, list.Data, 1)
		assert.Equal(t, map[string]string{first.chargeID: first.customer.ID}, chargeCustomerIDsByChargeID(t, list.Data))

		// The row is the same wire shape the customer-scoped list serves.
		flatFee, err := list.Data[0].AsChargeFlatFee()
		require.NoError(t, err)
		assert.Equal(t, v3sdk.ChargeStatusCreated, flatFee.Status)
		assertOutstandingRealization(t, flatFee.Realizations, flatFee.ServicePeriod, false)
	})

	t.Run("filter by customer_id oeq spans both customers", func(t *testing.T) {
		list, err := c.Charges.List(t.Context(), v3sdk.ListChargesParams{
			Filter: &v3sdk.ListChargesFilter{
				CustomerID: &v3sdk.StringExactFilter{Oeq: []string{first.customer.ID, second.customer.ID}},
			},
			Sort: &v3sdk.Sort{By: "created_at", Order: v3sdk.SortOrderDesc},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		assert.Equal(t, 2, list.Meta.Page.Total)
		assert.Equal(t, map[string]string{
			first.chargeID:  first.customer.ID,
			second.chargeID: second.customer.ID,
		}, chargeCustomerIDsByChargeID(t, list.Data))
	})

	t.Run("expand=customer resolves each row's own customer", func(t *testing.T) {
		list, err := c.Charges.List(t.Context(), v3sdk.ListChargesParams{
			Filter: &v3sdk.ListChargesFilter{
				CustomerID: &v3sdk.StringExactFilter{Oeq: []string{first.customer.ID, second.customer.ID}},
			},
			Expand: []v3sdk.ChargesExpand{v3sdk.ChargesExpandCustomer},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)
		require.Len(t, list.Data, 2)

		namesByCustomerID := map[string]string{
			first.customer.ID:  first.customer.Name,
			second.customer.ID: second.customer.Name,
		}
		for _, item := range list.Data {
			flatFee, err := item.AsChargeFlatFee()
			require.NoError(t, err)
			expanded, err := flatFee.Customer.AsCustomer()
			require.NoError(t, err)
			assert.Equal(t, namesByCustomerID[expanded.ID], expanded.Name, "charge %s carries a foreign customer", flatFee.ID)
		}
	})

	t.Run("pagination applies to the filtered set", func(t *testing.T) {
		list, err := c.Charges.List(t.Context(), v3sdk.ListChargesParams{
			Filter: &v3sdk.ListChargesFilter{
				CustomerID: &v3sdk.StringExactFilter{Oeq: []string{first.customer.ID, second.customer.ID}},
			},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(1), Number: lo.ToPtr(2)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		assert.Equal(t, 2, list.Meta.Page.Total)
		assert.Equal(t, 2, list.Meta.Page.Number)
		assert.Equal(t, 1, list.Meta.Page.Size)
		require.Len(t, list.Data, 1)
	})

	t.Run("the unfiltered list covers the whole namespace", func(t *testing.T) {
		list, err := c.Charges.List(t.Context(), v3sdk.ListChargesParams{
			Sort: &v3sdk.Sort{By: "created_at", Order: v3sdk.SortOrderDesc},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(1000)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		byCharge := chargeCustomerIDsByChargeID(t, list.Data)
		assert.Equal(t, first.customer.ID, byCharge[first.chargeID])
		assert.Equal(t, second.customer.ID, byCharge[second.chargeID])
	})
}
