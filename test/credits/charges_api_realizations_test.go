package credits

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	chargeshandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/charges"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// wireCharge is the subset of the v3 charge wire payload this suite asserts on.
// Decoding through a minimal struct keeps the assertions anchored to the raw
// JSON contract (snake_case field names) rather than to SDK conveniences.
type wireCharge struct {
	ID            string                         `json:"id"`
	Type          string                         `json:"type"`
	Status        string                         `json:"status"`
	ServicePeriod api.ClosedPeriod               `json:"service_period"`
	Realizations  []api.BillingChargeRealization `json:"realizations"`
	Usage         *string                        `json:"usage"`
}

// mustListChargeWireOverHTTP serves a real list-customer-charges request through
// the production v3 HTTP handler (router mounting mirrors
// api/v3/server/routes.go) and returns the single charge owned by the customer.
func (s *CreditThenInvoiceTestSuite) mustListChargeWireOverHTTP(ns string, customerID string, expands ...api.BillingChargesExpand) wireCharge {
	s.T().Helper()

	handler := chargeshandler.New(
		func(ctx context.Context) (string, error) { return ns, nil },
		s.Charges,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/customers/"+customerID+"/charges", nil)

	params := api.ListCustomerChargesParams{}
	if len(expands) > 0 {
		params.Expand = &expands
	}

	handler.ListCustomerCharges().With(chargeshandler.ListCustomerChargesParams{
		CustomerID: customerID,
		Params:     params,
	}).ServeHTTP(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	// The required realizations array must never serialize as JSON null.
	s.Contains(rec.Body.String(), `"realizations":[`)
	s.NotContains(rec.Body.String(), `"realizations":null`)

	var envelope struct {
		Data []wireCharge `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &envelope))
	s.Require().Len(envelope.Data, 1)

	return envelope.Data[0]
}

func (s *CreditThenInvoiceTestSuite) requireWireRealization(realization api.BillingChargeRealization, realizationType api.BillingChargeRealizationType, servicePeriod timeutil.ClosedPeriod, usage string) {
	s.T().Helper()

	s.Equal(realizationType, realization.Type)
	s.True(servicePeriod.From.Equal(realization.ServicePeriod.From), "service_period.from: want %v, got %v", servicePeriod.From, realization.ServicePeriod.From)
	s.True(servicePeriod.To.Equal(realization.ServicePeriod.To), "service_period.to: want %v, got %v", servicePeriod.To, realization.ServicePeriod.To)
	s.Require().NotNil(realization.Usage, "usage-based realizations always carry a usage field")
	s.Equal(usage, *realization.Usage)
}

// TestUsageBasedRealizationsWireShapeAcrossLifecycle drives a single
// credit-then-invoice usage charge through voiding, final realization, partial
// reclassification, and tail realization, asserting the realizations array on
// the v3 list wire payload at every stage. This is the multi-realization
// counterpart of the e2e outstanding-only smoke test
// (e2e/customer_charges_v3_test.go), which cannot reach these states because
// nothing in the e2e stack advances charges.
func (s *CreditThenInvoiceTestSuite) TestUsageBasedRealizationsWireShapeAcrossLifecycle() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-api-realizations-wire")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	firstExtendedTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	secondExtendedTo := datetime.MustParseTimeInLocation(t, "2026-04-01T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		voidedRunID        string
		partialRunID       string
	)

	s.Run("given a fresh charge, the wire carries only the outstanding projection", func() {
		// given:
		// - a ledger-backed customer holds 10 USD promotional credits, enough to
		//   fully credit every later invoice so the no-fiat path keeps payment
		//   processing out of scope
		// - 5 usage units are visible inside the original service period
		// when:
		// - a unit-priced credit-then-invoice usage charge is created
		// then:
		// - the list payload shows a single outstanding projection over the whole
		//   service period, with no persisted identity
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(10),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-api-realizations-wire",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-api-realizations-wire",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		wire := s.mustListChargeWireOverHTTP(ns, cust.ID)
		s.Equal(usageBasedChargeID.ID, wire.ID)
		s.Equal(string(api.BillingChargeUsageBasedTypeUsageBased), wire.Type)
		s.Equal(string(api.BillingChargeStatusCreated), wire.Status)

		s.Require().Len(wire.Realizations, 1)
		s.requireWireRealization(wire.Realizations[0], api.BillingChargeRealizationTypeOutstanding, servicePeriod, "0")
		s.Nil(wire.Realizations[0].Id)
		s.Nil(wire.Realizations[0].LineId)
		s.Nil(wire.Realizations[0].Invoice)
	})

	s.Run("when a mutable draft run is voided by an extension, the wire marks it voided", func() {
		// given:
		// - the pending line is collected into a mutable draft invoice, booking a
		//   current realization run
		// when:
		// - the charge is extended while the standard line is still mutable, which
		//   soft-deletes the line and voids the run
		// then:
		// - the voided run stays visible as an inert `voided` audit entry keeping
		//   its original period and usage, while the whole extended period is
		//   projected as outstanding (voided history covers nothing)
		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		clock.FreezeTime(invoices[0].DefaultCollectionAtForStandardInvoice())
		invoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveRealizationProcessing)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		voidedRunID = currentRun.ID.ID

		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, firstExtendedTo)
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		wire := s.mustListChargeWireOverHTTP(ns, cust.ID)
		s.Equal(string(api.BillingChargeStatusActive), wire.Status)

		s.Require().Len(wire.Realizations, 2)

		s.requireWireRealization(wire.Realizations[0], api.BillingChargeRealizationTypeVoided, servicePeriod, "5")
		s.Equal(lo.ToPtr(voidedRunID), wire.Realizations[0].Id, "the voided run keeps its identity on the wire")

		s.requireWireRealization(wire.Realizations[1], api.BillingChargeRealizationTypeOutstanding, timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   firstExtendedTo,
		}, "0")
		s.Nil(wire.Realizations[1].Id)
	})

	s.Run("when the extended period is invoiced to its end, the wire carries a final realization", func() {
		// given:
		// - 3 more usage units land inside the extension, for a cumulative 8
		// when:
		// - the gathering line is collected, advanced, and approved; full credit
		//   coverage finalizes the charge without a fiat transaction
		// then:
		// - the payload carries the voided audit entry plus one final_realization
		//   run over the whole extended period with the cumulative usage, and a
		//   final charge projects no outstanding tail
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			3,
			datetime.MustParseTimeInLocation(t, "2026-02-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(firstExtendedTo.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(firstExtendedTo),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		clock.FreezeTime(invoices[0].DefaultCollectionAtForStandardInvoice())
		invoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
		s.NoError(err)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.True(invoice.StatusDetails.Immutable)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusFinal)

		wire := s.mustListChargeWireOverHTTP(ns, cust.ID)
		s.Equal(string(api.BillingChargeStatusFinal), wire.Status)

		s.Require().Len(wire.Realizations, 2)

		s.requireWireRealization(wire.Realizations[0], api.BillingChargeRealizationTypeVoided, servicePeriod, "5")
		s.Equal(lo.ToPtr(voidedRunID), wire.Realizations[0].Id)

		s.requireWireRealization(wire.Realizations[1], api.BillingChargeRealizationTypeFinalRealization, timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   firstExtendedTo,
		}, "8")
		s.Require().NotNil(wire.Realizations[1].Id)
		partialRunID = *wire.Realizations[1].Id
		s.NotEqual(voidedRunID, partialRunID, "the voided run must not resurface as the booked run")
		s.NotNil(wire.Realizations[1].LineId)
		s.Require().NotNil(wire.Realizations[1].Invoice)
		invoiceRef, err := wire.Realizations[1].Invoice.AsChargeRealizationInvoiceReference()
		s.NoError(err)
		s.Equal(invoice.ID, invoiceRef.Id)
	})

	s.Run("when the final charge is extended, the run is reclassified as partial on the wire", func() {
		// given:
		// - the final run is backed by an immutable, fully credited invoice line
		// when:
		// - the charge is extended past the realized period
		// then:
		// - the same run resurfaces as partial_invoice and the uncovered tail is
		//   projected as outstanding
		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, secondExtendedTo)
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		wire := s.mustListChargeWireOverHTTP(ns, cust.ID)
		s.Equal(string(api.BillingChargeStatusActive), wire.Status)

		s.Require().Len(wire.Realizations, 3)

		s.requireWireRealization(wire.Realizations[0], api.BillingChargeRealizationTypeVoided, servicePeriod, "5")

		s.requireWireRealization(wire.Realizations[1], api.BillingChargeRealizationTypePartialInvoice, timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   firstExtendedTo,
		}, "8")
		s.Equal(lo.ToPtr(partialRunID), wire.Realizations[1].Id, "reclassification must keep the run identity")

		s.requireWireRealization(wire.Realizations[2], api.BillingChargeRealizationTypeOutstanding, timeutil.ClosedPeriod{
			From: firstExtendedTo,
			To:   secondExtendedTo,
		}, "0")
		s.Nil(wire.Realizations[2].Id)
	})

	s.Run("when the tail is invoiced, the wire carries partial and final runs with per-run usage", func() {
		// given:
		// - 2 more usage units land inside the second extension, for a cumulative 10
		// when:
		// - the tail gathering line is collected, advanced, and approved
		// then:
		// - the payload carries the partial run and the new final run stitched into
		//   contiguous service periods, each reporting its own usage rather than
		//   the cumulative quantity, and no outstanding remains
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			2,
			datetime.MustParseTimeInLocation(t, "2026-03-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(secondExtendedTo.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(secondExtendedTo),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		clock.FreezeTime(invoices[0].DefaultCollectionAtForStandardInvoice())
		invoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
		s.NoError(err)

		_, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusFinal)

		wire := s.mustListChargeWireOverHTTP(ns, cust.ID)
		s.Equal(string(api.BillingChargeStatusFinal), wire.Status)

		s.Require().Len(wire.Realizations, 3)

		s.requireWireRealization(wire.Realizations[0], api.BillingChargeRealizationTypeVoided, servicePeriod, "5")

		s.requireWireRealization(wire.Realizations[1], api.BillingChargeRealizationTypePartialInvoice, timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   firstExtendedTo,
		}, "8")
		s.Equal(lo.ToPtr(partialRunID), wire.Realizations[1].Id)

		s.requireWireRealization(wire.Realizations[2], api.BillingChargeRealizationTypeFinalRealization, timeutil.ClosedPeriod{
			From: firstExtendedTo,
			To:   secondExtendedTo,
		}, "2")
		s.Require().NotNil(wire.Realizations[2].Id)
		s.NotEqual(partialRunID, *wire.Realizations[2].Id)
		s.NotEqual(voidedRunID, *wire.Realizations[2].Id)
	})

	s.Run("expands hydrate invoice headers and the live usage on the wire", func() {
		// given:
		// - the finalized charge carries a voided run plus two booked runs, each
		//   realized into an approved invoice
		// when:
		// - the wire is read with the realization.invoice and real_time_usage expands
		// then:
		// - the charge-level usage reports the live cumulative read (5+3+2)
		// - every booked realization embeds the invoice header of its booked
		//   line: the invoice entity without its lines
		wire := s.mustListChargeWireOverHTTP(ns, cust.ID,
			api.BillingChargesExpandRealizationInvoice,
			api.BillingChargesExpandRealTimeUsage,
		)

		s.Require().NotNil(wire.Usage, "real_time_usage fills the charge-level usage")
		s.Equal("10", *wire.Usage)

		s.Require().Len(wire.Realizations, 3)
		for _, idx := range []int{1, 2} {
			s.Require().NotNil(wire.Realizations[idx].Invoice, "realization %d", idx)

			header, err := wire.Realizations[idx].Invoice.AsBillingChargeRealizationInvoice()
			s.Require().NoError(err, "realization %d carries the embedded invoice header branch", idx)
			s.NotEmpty(header.Id)
		}
	})
}
