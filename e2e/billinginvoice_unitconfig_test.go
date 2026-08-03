package e2e

import (
	"net/http"
	"slices"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestInvoiceUnitConfigConvertedQuantity drives the full unit_config billing
// pipeline against the live e2e stack and asserts that a usage-based invoice
// line records the raw metered quantity alongside the unit-config-converted
// invoiced quantity.
//
// Scenario: a SUM meter over `$.value`, billed per 1000 units rounded up
// (unit_config: divide by 1000, ceiling). Ingesting a single 7400 event yields
// meteredQuantity=7400 (raw) and quantity=ceil(7400/1000)=8 (converted).
func TestInvoiceUnitConfigConvertedQuantity(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	uniq := uniqueKey("ucbill")
	meterKey := "uc_bill_meter_" + uniq
	eventType := "uc_bill_event_" + uniq
	featureKey := "uc_bill_feature_" + uniq
	customerKey := "uc_bill_customer_" + uniq
	subjectKey := "uc_bill_subject_" + uniq

	var meter *v3sdk.Meter
	var feature *v3sdk.Feature
	var plan *v3sdk.Plan
	var customer *api.Customer
	var subscription *v3sdk.BillingSubscription

	// given:
	// - a SUM meter over $.value and a feature bound to it
	// - a published plan whose single usage-based rate card carries a
	//   divide-by-1000 ceiling unit_config
	// when:
	// - the catalog fixtures are created via the v3 API
	// then:
	// - the plan publishes and is ready for subscription
	runRequired(t, "creates unit_config catalog fixtures", func(t *testing.T) {
		createdMeter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:           meterKey,
			Name:          "Unit Config Meter " + uniq,
			Aggregation:   v3sdk.MeterAggregationSum,
			EventType:     eventType,
			ValueProperty: lo.ToPtr("$.value"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdMeter)
		meter = createdMeter

		createdFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   featureKey,
			Name:  "Unit Config Feature " + uniq,
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdFeature)
		feature = createdFeature

		// Unit price billed per converted unit. Unit prices require an
		// in-arrears payment term; unit_config divides raw usage by 1000 and
		// rounds the invoiced quantity up to whole units.
		cadence := "P1M"
		term := v3sdk.PricePaymentTermInArrears
		price := lo.Must(v3sdk.PriceFromPriceUnit(v3sdk.PriceUnit{
			Amount: "0.10",
		}))

		rateCard := v3sdk.RateCardInput{
			Key:            feature.Key,
			Name:           "Unit Config Rate Card " + uniq,
			Price:          price,
			BillingCadence: &cadence,
			PaymentTerm:    &term,
			Feature:        &v3sdk.FeatureReference{ID: feature.ID},
			UnitConfig: &v3sdk.UnitConfig{
				Operation:        v3sdk.UnitConfigOperationDivide,
				ConversionFactor: "1000",
				Rounding:         lo.ToPtr(v3sdk.UnitConfigRoundingModeCeiling),
				Precision:        lo.ToPtr(int64(0)),
			},
		}

		createdPlan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            "uc_bill_plan_" + uniq,
			Name:           "Unit Config Plan " + uniq,
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       "phase_1",
				Name:      "Unit Config Phase",
				RateCards: []v3sdk.RateCardInput{rateCard},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdPlan)

		plan, err = c.Plans.Publish(t.Context(), createdPlan.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		require.Equal(t, v3sdk.PlanStatusActive, plan.Status)
	})

	// given:
	// - a customer with usage attribution to the ingested subject key
	// - a published unit_config plan
	// when:
	// - the customer subscribes in credit-then-invoice mode
	// then:
	// - the subscription is active and exposes its billing anchor
	runRequired(t, "creates customer and starts subscription", func(t *testing.T) {
		// The v3 create-customer request does not model usage-attribution
		// subject keys, so the customer (and its subject) is provisioned via
		// the v1 SDK helper to bind the ingested subject to the customer.
		customer = CreateCustomerWithSubject(t, v1, customerKey, subjectKey)
		require.NotNil(t, customer)

		createdSubscription, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:       v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customer.Id)},
			Plan:           v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(plan.ID)},
			SettlementMode: lo.ToPtr(v3sdk.SettlementModeCreditThenInvoice),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdSubscription)
		require.Equal(t, v3sdk.SubscriptionStatusActive, createdSubscription.Status)
		subscription = createdSubscription
	})

	// given:
	// - an active subscription with a known billing anchor
	// when:
	// - a single event carrying value=7400 is ingested at the billing anchor
	//   so the usage lands inside the in-arrears period
	// then:
	// - the (asynchronous) sink eventually aggregates the meter to 7400
	runRequired(t, "ingests usage and waits for the meter to aggregate", func(t *testing.T) {
		ev := cloudevents.New()
		ev.SetID(uniqueKey("uc_bill_evt"))
		ev.SetSource("uc-bill-e2e")
		ev.SetType(eventType)
		ev.SetSubject(subjectKey)
		ev.SetTime(subscription.BillingAnchor)
		require.NoError(t, ev.SetData("application/json", map[string]string{
			"value": "7400",
		}))

		resp, err := v1.IngestEventWithResponse(t.Context(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode(), "ingest event: %s", string(resp.Body))

		// The sink worker processes events asynchronously, so poll the meter
		// until the single ingested value is visible.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			result, err := queryMeterV3(t, meter.ID, v3sdk.MeterQueryRequest{})
			require.NoError(collect, err)
			require.NotNil(collect, result)
			require.NotEmpty(collect, result.Data)
			require.Equal(collect, float64(7400), numericToFloat(collect, result.Data[0].Value))
		}, time.Minute, time.Second)
	})

	// given:
	// - aggregated usage of 7400 on an in-arrears subscription
	// when:
	// - the subscription is canceled, closing the period so the billing-worker
	//   produces an invoice, and the usage-based line's quantities are snapshotted
	// then:
	// - the usage line records meteredQuantity=7400 (raw) and quantity=8
	//   (ceil(7400/1000)) from the unit_config conversion
	runRequired(t, "cancels the subscription and asserts converted invoice quantity", func(t *testing.T) {
		canceledSubscription, err := c.Subscriptions.Cancel(t.Context(), subscription.ID, v3sdk.SubscriptionCancel{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, canceledSubscription)
		require.Equal(t, v3sdk.SubscriptionStatusInactive, canceledSubscription.Status)

		usageLine := waitForSnapshottedUsageLine(t, v1, customer.Id, featureKey)

		require.NotNil(t, usageLine.MeteredQuantity, "usage line metered quantity")
		require.NotNil(t, usageLine.Quantity, "usage line invoiced quantity")
		assert.Equal(t, float64(7400), numericToFloat(t, *usageLine.MeteredQuantity), "raw metered quantity")
		assert.Equal(t, float64(8), numericToFloat(t, *usageLine.Quantity), "unit-config-converted invoiced quantity")
	})
}

func waitForSnapshottedUsageLine(t *testing.T, client *api.ClientWithResponses, customerID, featureKey string) api.InvoiceLine {
	t.Helper()

	ctx := t.Context()
	customers := api.InvoiceListParamsCustomers{customerID}
	expand := api.InvoiceListParamsExpand{api.InvoiceExpandLines}
	snapshotRequested := map[string]bool{}
	var usageLine api.InvoiceLine

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := client.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
			Customers: &customers,
			Expand:    &expand,
			PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
		})
		require.NoError(collect, err)
		require.Equal(collect, http.StatusOK, resp.StatusCode(), "list invoices: %s", string(resp.Body))
		require.NotNil(collect, resp.JSON200)

		for _, invoice := range resp.JSON200.Items {
			if invoice.Lines == nil {
				continue
			}

			idx := slices.IndexFunc(*invoice.Lines, func(line api.InvoiceLine) bool {
				return line.FeatureKey != nil && *line.FeatureKey == featureKey
			})
			if idx == -1 {
				continue
			}

			line := (*invoice.Lines)[idx]
			if line.MeteredQuantity != nil && line.Quantity != nil {
				usageLine = line
				return
			}

			if invoice.StatusDetails.AvailableActions.SnapshotQuantities != nil && !snapshotRequested[invoice.Id] {
				snapshotResp, err := client.SnapshotQuantitiesInvoiceActionWithResponse(ctx, invoice.Id)
				require.NoError(collect, err)
				require.Equal(collect, http.StatusOK, snapshotResp.StatusCode(), "snapshot quantities: %s", string(snapshotResp.Body))
				snapshotRequested[invoice.Id] = true
			}
		}

		require.Fail(collect, "usage-based invoice line with snapshotted quantities not found yet")
	}, 2*time.Minute, time.Second)

	return usageLine
}
