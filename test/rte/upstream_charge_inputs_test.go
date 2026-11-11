package rte_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargesservice "github.com/openmeterio/openmeter/openmeter/billing/charges/service"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

func (s *RTEIntegrationSuite) TestPendingPurchaseInputFactRefreshesCreditsBeforePosting() {
	// given: an active credit calculation and the root charge service publishing
	// persisted input facts through the real transactional outbox.
	ctx := s.T().Context()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	ns := s.GetUniqueNamespace("rte-pending-input")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	app := s.SetupCustomInvoicing(ns)
	s.ProvisionBillingProfile(ctx, ns, app.App.GetID())
	cust := s.CreateLedgerBackedCustomer(ns, "rte-pending-input")
	pipeline := s.newEventPipeline()
	logger := testutils.NewDiscardLogger(s.T())
	adapter, err := chargesadapter.New(chargesadapter.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	metaAdapter, err := metaadapter.New(metaadapter.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	writer, err := chargesservice.New(chargesservice.Config{
		Logger: logger, Adapter: adapter, MetaAdapter: metaAdapter,
		FeatureMeterResolver: s.FeatureMeterResolver, FlatFeeService: s.FlatFeeSvc,
		CreditPurchaseService: s.CreditPurchaseSvc, UsageBasedService: s.UsageBasedSvc,
		RecognizerService: s.RevenueRecognizer, BillingService: s.BillingService,
		TaxCodeService: s.TaxCodeEnv.Service, CurrencyResolver: s.CurrencyResolver,
		CustomerService: s.CustomerService, SubscriptionService: &chargestestutils.FakeSubscriptionService{},
		InputChangePublisher: pipeline.publisher,
	})
	s.Require().NoError(err)
	key := creditbalance.Key{CustomerID: cust.GetID(), Currency: currencies.NewCurrencyReference("USD")}
	target, err := rte.NewCalculationRequestedEvent(key)
	s.Require().NoError(err)
	demands, err := common.NewRTEDemandService(s.DBClient, pipeline.service, logger, metricnoop.NewMeterProvider().Meter("test"), config.RTEConfiguration{CalculationTimeout: 30 * time.Second})
	s.Require().NoError(err)
	s.Require().NoError(demands.Activate(ctx, rte.ActivateDemandInput{Target: target.CalculationTarget}))
	defer func() {
		s.Require().NoError(demands.Deactivate(ctx, target.CalculationTarget))
	}()
	s.Require().NoError(demands.EnsureFresh(ctx, rte.EnsureFreshInput{Target: target.CalculationTarget, PullTTL: time.Minute}))
	baseline, err := pipeline.service.CreditBalances().Get(ctx, key)
	s.Require().NoError(err)
	s.Zero(baseline.Value.Pending.InexactFloat64())
	period := timeutil.ClosedPeriod{From: start.Add(24 * time.Hour), To: start.AddDate(0, 1, 0)}
	input := charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(charges.NewChargeIntent(creditpurchase.Intent{
		Intent: meta.Intent{ManagedBy: billing.ManuallyManagedLine, CustomerID: cust.ID, Currency: currenciestestutils.NewFiatCurrency(s.T(), "USD")},
		IntentMutableFields: creditpurchase.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{Name: "Scheduled purchase", ServicePeriod: period, BillingPeriod: period, FullServicePeriod: period},
			CreditAmount:        alpacadecimal.NewFromInt(25), Settlement: creditpurchase.NewInvoiceSettlement(),
		},
		CostBasis: creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: alpacadecimal.NewFromInt(1)}),
	}))}
	clock.FreezeTime(start.Add(time.Second))
	defer clock.UnFreeze()
	rollback := errors.New("abort charge creation")

	// when: creation is rolled back, both the input and its notification disappear.
	err = transaction.RunWithNoValue(ctx, enttx.NewCreator(s.DBClient), func(ctx context.Context) error {
		_, err := writer.Create(ctx, input)
		if err != nil {
			return err
		}

		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return err
		}

		client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
		count, err := client.EventOutbox.Query().Where(eventoutbox.TransactionIDEQ(tx.ID())).Count(ctx)
		s.Require().NoError(err)
		s.Equal(1, count)

		return rollback
	})
	s.Require().ErrorIs(err, rollback)
	count, err := s.DBClient.ChargeCreditPurchase.Query().Where(chargecreditpurchase.NamespaceEQ(ns)).Count(ctx)
	s.Require().NoError(err)
	s.Zero(count)
	_, err = writer.Create(ctx, input)
	s.Require().NoError(err)

	// then: the committed input fact refreshes pending credit while no ledger
	// transaction group exists for this customer yet.
	seenFact, seenPending := false, false
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for !seenFact || !seenPending {
		select {
		case message := <-pipeline.messages:
			message.Ack()
			switch pipeline.publisher.Marshaler().NameFromMessage(message) {
			case (charges.ChargeInputsChangedEvent{}).EventName():
				var fact charges.ChargeInputsChangedEvent
				s.Require().NoError(pipeline.publisher.Marshaler().Unmarshal(message, &fact))
				s.Equal(cust.ID, fact.CustomerID)
				seenFact = true
			case (rte.CalculationUpdatedEvent{}).EventName():
				var updated rte.CalculationUpdatedEvent
				s.Require().NoError(pipeline.publisher.Marshaler().Unmarshal(message, &updated))
				var state rte.State[creditbalance.Balance]
				s.Require().NoError(json.Unmarshal(updated.State, &state))
				seenPending = seenPending || state.Value.Pending.InexactFloat64() == 25
			}
		case <-timer.C:
			s.FailNow("pending credit update was not delivered")
		}
	}

	groups, err := s.DBClient.LedgerTransactionGroup.Query().Where(ledgertransactiongroup.NamespaceEQ(ns)).Count(ctx)
	s.Require().NoError(err)
	s.Zero(groups)
	stored, err := pipeline.service.CreditBalances().Get(ctx, key)
	s.Require().NoError(err)
	s.Equal(float64(25), stored.Value.Pending.InexactFloat64())
	s.Zero(stored.Value.Booked.InexactFloat64())

	// Patching a future credit-only charge uses the same atomic input boundary,
	// even though neither creation nor deletion needs a ledger posting yet.
	created, err := writer.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(
		s.CreateMockChargeIntent(creditstest.CreateMockChargeIntentInput{
			Customer: cust.GetID(), Currency: creditstest.USD, ServicePeriod: period,
			Price:          productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(2), PaymentTerm: productcatalog.InAdvancePaymentTerm}),
			SettlementMode: productcatalog.CreditOnlySettlementMode, Name: "Scheduled fee", ManagedBy: billing.ManuallyManagedLine,
		}),
	)})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	chargeID, err := created[0].GetChargeID()
	s.Require().NoError(err)
	patch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Policy:       meta.PatchDeletePolicy{CreditRefundPolicy: meta.CreditRefundPolicyIgnore, InvoiceRefundPolicy: meta.InvoiceRefundPolicyIgnore},
	})
	s.Require().NoError(err)
	err = transaction.RunWithNoValue(ctx, enttx.NewCreator(s.DBClient), func(ctx context.Context) error {
		if err := writer.ApplyPatches(ctx, charges.ApplyPatchesInput{CustomerID: cust.GetID(), PatchesByChargeID: map[string]charges.Patch{chargeID.ID: patch}}); err != nil {
			return err
		}

		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return err
		}

		client := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()
		count, err := client.EventOutbox.Query().Where(eventoutbox.TransactionIDEQ(tx.ID())).Count(ctx)
		s.Require().NoError(err)
		s.Equal(1, count)

		return rollback
	})
	s.Require().ErrorIs(err, rollback)
	retained, err := writer.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
	s.Require().NoError(err)
	fee, err := retained.AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Nil(fee.DeletedAt)
}
