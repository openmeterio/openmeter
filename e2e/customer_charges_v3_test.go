package e2e

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// assertOutstandingRealization asserts realizations carries exactly one entry: a
// projected `outstanding` run spanning the charge's whole service period. A
// freshly created charge has booked no realization run yet, so the outstanding
// entry built inline in the realization converters (convertFlatFeeRealizationsToAPI /
// convertUsageBasedRealizationsToAPI in api/v3/handlers/customers/charges/convert.go)
// always reports the entire, still-uncovered span with no persisted identity.
func assertOutstandingRealization(t *testing.T, realizations []v3sdk.ChargeRealization, servicePeriod v3sdk.ClosedPeriod, metered bool) {
	t.Helper()
	require.Len(t, realizations, 1)

	r := realizations[0]
	assert.Equal(t, v3sdk.ChargeRealizationTypeOutstanding, r.Type)
	assert.Nil(t, r.ID)
	assert.Nil(t, r.LineID)
	assert.Nil(t, r.Invoice)
	assert.Nil(t, r.Payment)
	assert.True(t, servicePeriod.From.Equal(r.ServicePeriod.From), "service_period.from: want %v, got %v", servicePeriod.From, r.ServicePeriod.From)
	assert.True(t, servicePeriod.To.Equal(r.ServicePeriod.To), "service_period.to: want %v, got %v", servicePeriod.To, r.ServicePeriod.To)

	if !metered {
		// Flat fees are not metered, so their realizations carry no usage
		// field at all.
		assert.Nil(t, r.Usage)
		return
	}

	// The outstanding entry's usage is zero here because no real_time_usage
	// expand is applied; under that expand it reports the not-yet-booked
	// remainder of a live metering read (the outstanding entry built inline
	// in convertUsageBasedRealizationsToAPI, convert.go). Parse rather than
	// comparing against the raw string, since decimals are normalized.
	require.NotNil(t, r.Usage, "usage-based realizations always carry a usage field")
	usage, err := strconv.ParseFloat(*r.Usage, 64)
	require.NoError(t, err)
	assert.Zero(t, usage)
}

// TestV3CustomerChargeFlatFeeRealizations verifies that a freshly created flat
// fee charge carries exactly one `outstanding` realization spanning its full
// service period, on both the create response and the subsequent list read,
// and that the `realizations` field never serializes as JSON null - it is a
// required (non-nullable) field per the charges.tsp contract.
func TestV3CustomerChargeFlatFeeRealizations(t *testing.T) {
	c := newV3Client(t)
	prefix := uniqueKey("charge_flatfee")

	// given:
	// - a USD customer to own the charge
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey(prefix + "_customer"),
		Name:     "Charge Flat Fee Test Customer",
		Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	// A service period with a real, bounded span: the outstanding realization
	// entry only fills in when covered-until is strictly before
	// service_period.to, so a zero-length period would test nothing.
	servicePeriod := v3sdk.ClosedPeriod{
		From: time.Now().UTC().Truncate(time.Second),
		To:   time.Now().UTC().Truncate(time.Second).Add(30 * 24 * time.Hour),
	}

	createReq, err := v3sdk.CreateChargeRequestFromCreateChargeFlatFeeRequest(v3sdk.CreateChargeFlatFeeRequest{
		Name:           "Flat Fee " + prefix,
		Type:           v3sdk.ChargeTypeFlatFee,
		Currency:       "USD",
		InvoiceAt:      servicePeriod.To,
		ServicePeriod:  servicePeriod,
		SettlementMode: v3sdk.SettlementModeCreditThenInvoice,
		PaymentTerm:    v3sdk.PricePaymentTermInArrears,
		ProrationConfiguration: v3sdk.RateCardProrationConfiguration{
			Mode: v3sdk.RateCardProrationModeNoProration,
		},
		// amount_before_proration.currency must equal the top-level currency, or
		// the server rejects the request with a 400
		// (fromAPICreateChargeFlatFeeRequest in customers/charges/convert.go).
		AmountBeforeProration: v3sdk.CurrencyAmount{Amount: "10", Currency: "USD"},
	})
	require.NoError(t, err)

	// when:
	// - the charge is created over HTTP
	created, err := c.Customers.Charges.Create(t.Context(), customer.ID, createReq)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, created)
	require.Equal(t, string(v3sdk.ChargeTypeFlatFee), created.Type)

	flatFee, err := created.AsChargeFlatFee()
	require.NoError(t, err)
	require.NotEmpty(t, flatFee.ID)
	chargeID := flatFee.ID

	t.Run("create response reflects the charge and carries the outstanding realization", func(t *testing.T) {
		// then:
		// - charge-level fields reflect what was sent
		assert.Equal(t, v3sdk.ChargeStatusCreated, flatFee.Status)
		customerRef, err := flatFee.Customer.AsCustomerReference()
		require.NoError(t, err)
		assert.Equal(t, customer.ID, customerRef.ID)
		assert.Equal(t, "USD", flatFee.Currency)
		assert.Equal(t, v3sdk.SettlementModeCreditThenInvoice, flatFee.SettlementMode)

		// ConvertDecimalToCurrencyAmount (api/v3/handlers/customers/charges/convert.go)
		// never sets Currency on amount_after_proration, so only the amount is
		// asserted here.
		amount, err := strconv.ParseFloat(flatFee.AmountAfterProration.Amount, 64)
		require.NoError(t, err)
		assert.Equal(t, float64(10), amount)

		// - realizations carries exactly the outstanding projection
		assertOutstandingRealization(t, flatFee.Realizations, flatFee.ServicePeriod, false)
	})

	t.Run("list response carries the same realization shape", func(t *testing.T) {
		// when:
		// - the customer's charges are listed; the listing is scoped to this
		//   fresh customer, so the default page size always covers it
		list, err := c.Customers.Charges.List(t.Context(), customer.ID, v3sdk.ChargeListParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		var listed *v3sdk.ChargeFlatFee
		for _, ch := range list.Data {
			if ch.Type != string(v3sdk.ChargeTypeFlatFee) {
				continue
			}
			candidate, err := ch.AsChargeFlatFee()
			require.NoError(t, err)
			if candidate.ID == chargeID {
				listed = candidate
				break
			}
		}
		require.NotNil(t, listed, "charge %s not found in list", chargeID)

		// then:
		// - the listed copy carries the same realization shape as the create
		//   response - this is the converter path under test
		assertOutstandingRealization(t, listed.Realizations, listed.ServicePeriod, false)
	})

	t.Run("realizations never serializes as null on the raw wire body", func(t *testing.T) {
		// realizations is a required (non-optional) array in charges.tsp; assert
		// the wire contract directly rather than only through the typed SDK,
		// which would silently unmarshal a JSON null into an empty slice.
		status, raw, problem := c.doMalformedRequest(http.MethodGet, "/customers/"+customer.ID+"/charges", nil)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)

		body := string(raw)
		assert.Contains(t, body, `"realizations":[`)
		assert.NotContains(t, body, `"realizations":null`)
	})
}

// TestV3CustomerChargeUsageBasedRealizations verifies the same outstanding
// realization shape as TestV3CustomerChargeFlatFeeRealizations, but for a
// usage-based charge, and additionally asserts that the feature and price
// references round-trip through create and list.
func TestV3CustomerChargeUsageBasedRealizations(t *testing.T) {
	c := newV3Client(t)
	prefix := uniqueKey("charge_usage")

	// given:
	// - a USD customer and a feature (with a backing meter, matching the
	//   established e2e fixture pattern) to own and reference the charge
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey(prefix + "_customer"),
		Name:     "Charge Usage Based Test Customer",
		Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
		Key:         prefix + "_meter",
		Name:        "Charge Usage Based Meter " + prefix,
		Aggregation: v3sdk.MeterAggregationCount,
		EventType:   prefix + "_event",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, meter)

	feature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:   prefix + "_feature",
		Name:  "Charge Usage Based Feature " + prefix,
		Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, feature)

	servicePeriod := v3sdk.ClosedPeriod{
		From: time.Now().UTC().Truncate(time.Second),
		To:   time.Now().UTC().Truncate(time.Second).Add(30 * 24 * time.Hour),
	}

	// A flat price keeps this smoke test free of meter setup; metered price
	// types (unit/graduated/volume) are accepted too via
	// fromAPIChargeBillingPrice (customers/charges/convert.go).
	price, err := v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{Amount: "1.5"})
	require.NoError(t, err)

	createReq, err := v3sdk.CreateChargeRequestFromCreateChargeUsageBasedRequest(v3sdk.CreateChargeUsageBasedRequest{
		Name:           "Usage Based " + prefix,
		Type:           v3sdk.ChargeTypeUsageBased,
		Currency:       "USD",
		InvoiceAt:      servicePeriod.To,
		ServicePeriod:  servicePeriod,
		SettlementMode: v3sdk.SettlementModeCreditThenInvoice,
		Price:          price,
		Feature:        v3sdk.FeatureReference{ID: feature.ID},
	})
	require.NoError(t, err)

	// when:
	// - the charge is created over HTTP
	created, err := c.Customers.Charges.Create(t.Context(), customer.ID, createReq)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, created)
	require.Equal(t, string(v3sdk.ChargeTypeUsageBased), created.Type)

	usageBased, err := created.AsChargeUsageBased()
	require.NoError(t, err)
	require.NotEmpty(t, usageBased.ID)
	chargeID := usageBased.ID

	t.Run("create response reflects the charge and carries the outstanding realization", func(t *testing.T) {
		// then:
		// - charge-level fields reflect what was sent
		assert.Equal(t, v3sdk.ChargeStatusCreated, usageBased.Status)
		customerRef, err := usageBased.Customer.AsCustomerReference()
		require.NoError(t, err)
		assert.Equal(t, customer.ID, customerRef.ID)
		assert.Equal(t, "USD", usageBased.Currency)
		assert.Equal(t, v3sdk.SettlementModeCreditThenInvoice, usageBased.SettlementMode)

		featureRef, err := usageBased.Feature.AsFeatureReference()
		require.NoError(t, err)
		assert.Equal(t, feature.ID, featureRef.ID)

		require.Equal(t, string(v3sdk.PriceTypeFlat), usageBased.Price.Type)
		flatPrice, err := usageBased.Price.AsPriceFlat()
		require.NoError(t, err)
		priceAmount, err := strconv.ParseFloat(flatPrice.Amount, 64)
		require.NoError(t, err)
		assert.Equal(t, 1.5, priceAmount)

		// - realizations carries exactly the outstanding projection
		assertOutstandingRealization(t, usageBased.Realizations, usageBased.ServicePeriod, true)
	})

	t.Run("list response carries the same realization shape", func(t *testing.T) {
		// when:
		// - the customer's charges are listed; the listing is scoped to this
		//   fresh customer, so the default page size always covers it
		list, err := c.Customers.Charges.List(t.Context(), customer.ID, v3sdk.ChargeListParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		var listed *v3sdk.ChargeUsageBased
		for _, ch := range list.Data {
			if ch.Type != string(v3sdk.ChargeTypeUsageBased) {
				continue
			}
			candidate, err := ch.AsChargeUsageBased()
			require.NoError(t, err)
			if candidate.ID == chargeID {
				listed = candidate
				break
			}
		}
		require.NotNil(t, listed, "charge %s not found in list", chargeID)

		// then:
		// - the listed copy carries the same realization shape as the create
		//   response - this is the converter path under test
		assertOutstandingRealization(t, listed.Realizations, listed.ServicePeriod, true)
	})
}
