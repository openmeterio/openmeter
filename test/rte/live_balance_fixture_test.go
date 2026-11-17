package rte_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

type liveBalanceFixture struct {
	customer    *customer.Customer
	entitlement *entitlement.Entitlement
	creditKey   creditbalance.Key
	meteredKey  rtemetered.Key
	meterID     string
	subjectKey  string
	period      timeutil.ClosedPeriod
}

func (s *RTELiveProcessSuite) setupLiveBalances() (liveBalanceFixture, func()) {
	t := s.T()
	t.Helper()

	ctx := t.Context()

	// Both balances use the same customer and meter. The charge prices each unit
	// at one USD, and the entitlement starts with a 100-unit grant.
	ns := s.GetUniqueNamespace("rte-live")
	const subjectKey = "rte-live-customer"
	cust := s.CreateLedgerBackedCustomer(ns, subjectKey)

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)

	s.TaxCodeEnv.ProvisionDefaultTaxCodes(t, ns)
	app := s.InstallSandboxApp(t, ns)
	s.ProvisionBillingProfile(ctx, ns, app.GetID())

	// Keep usage in completed minute windows for the metered calculator.
	start := time.Now().UTC().Truncate(time.Minute).Add(-5 * time.Minute)
	period := timeutil.ClosedPeriod{From: start, To: start.Add(24 * time.Hour)}

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.NewCreateChargeIntents(s.CreateMockChargeIntent(creditstest.CreateMockChargeIntentInput{
			Customer:          cust.GetID(),
			Currency:          creditstest.USD,
			ServicePeriod:     period,
			Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			FeatureKey:        feature.Feature.Key,
			Name:              "live usage",
			ManagedBy:         billing.SubscriptionManagedLine,
			SettlementMode:    productcatalog.CreditOnlySettlementMode,
			UniqueReferenceID: "live-usage",
		})),
	})
	require.NoError(t, err)

	measure := &entitlement.MeasureUsageFromInput{}
	require.NoError(t, measure.FromTime(start))

	ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
		CustomerID: cust.GetID(),
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID:        &feature.Feature.ID,
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: measure,
			IssueAfterReset:  lo.ToPtr(float64(100)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor:   start,
				Interval: timeutil.RecurrencePeriodMonth,
			})),
		},
	})
	require.NoError(t, err)

	currency := currencies.NewCurrencyReference(creditstest.USD)
	creditKey := creditbalance.Key{CustomerID: cust.GetID(), Currency: currency}
	meteredKey := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}

	return liveBalanceFixture{
		customer:    cust,
		entitlement: ent,
		creditKey:   creditKey,
		meteredKey:  meteredKey,
		meterID:     *feature.Feature.MeterID,
		subjectKey:  subjectKey,
		period:      period,
	}, feature.Cleanup
}

// fundLiveCredits issues and settles receivables so the balance contains funded credits.
// Passing the writer explicitly controls whether the posting emits a source event.
func (s *RTELiveProcessSuite) fundLiveCredits(ctx context.Context, writer ledger.Ledger, key creditbalance.Key, amount int64) {
	t := s.T()
	t.Helper()

	at := time.Now().UTC().Add(-time.Second)
	inputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
		AccountService: s.LedgerResolver,
		AccountCatalog: s.LedgerAccountService,
		BalanceQuerier: s.BalanceQuerier,
	}, transactions.ResolutionScope{CustomerID: key.CustomerID, Namespace: key.CustomerID.Namespace},
		transactions.IssueCustomerReceivableTemplate{At: at, Amount: alpacadecimal.NewFromInt(amount), Currency: key.Currency},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{At: at, Amount: alpacadecimal.NewFromInt(amount), Currency: key.Currency},
		transactions.SettleCustomerReceivableFromPaymentTemplate{At: at, Amount: alpacadecimal.NewFromInt(amount), Currency: key.Currency},
	)
	require.NoError(t, err)

	_, err = writer.CommitGroup(ctx, transactions.GroupInputs(key.CustomerID.Namespace, nil, inputs...))
	require.NoError(t, err)
}

// storeLateUsage commits 25 units in ClickHouse and prepares the sink notification.
// The caller publishes it explicitly so the test can also redeliver the same event.
func (s *RTELiveProcessSuite) storeLateUsage(live *liveProcess, fixture liveBalanceFixture) *message.Message {
	t := s.T()
	t.Helper()

	ctx := t.Context()

	eventTime := fixture.period.From.Add(time.Minute)
	storedAt := time.Now().UTC()
	raw := streaming.RawEvent{
		Namespace:  fixture.customer.Namespace,
		ID:         ulid.Make().String(),
		Type:       "test",
		Source:     "rte-live-test",
		Subject:    fixture.subjectKey,
		CustomerID: &fixture.customer.ID,
		Time:       eventTime,
		IngestedAt: storedAt,
		StoredAt:   storedAt,
		Data:       `{"value":25}`,
	}
	require.NoError(t, live.streaming.BatchInsert(ctx, []streaming.RawEvent{raw}))

	meterReader, err := meteradapter.New(meteradapter.Config{Client: s.DBClient, Logger: testutils.NewDiscardLogger(t)})
	require.NoError(t, err)

	definition, err := meterReader.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: fixture.customer.Namespace, IDOrSlug: fixture.meterID})
	require.NoError(t, err)

	rows, err := live.streaming.QueryMeter(ctx, fixture.customer.Namespace, definition, streaming.QueryParams{
		FilterCustomer: []streaming.Customer{fixture.customer},
		From:           &fixture.period.From,
		To:             &fixture.period.To,
	})
	require.NoError(t, err)

	var storedUsage float64
	for _, row := range rows {
		storedUsage += row.Value
	}

	require.Equal(t, float64(25), storedUsage, "persisted production meter must see inserted usage")
	t.Logf("inserted usage=%g meter=%s aggregation=%s eventType=%s topic=%s",
		storedUsage, definition.Key, definition.Aggregation, definition.EventType, live.conf.Events.IngestEvents.Topic)

	// The sink can store this usage before its meter cache picks up the new meter.
	notification := ingestevents.EventBatchedIngest{
		Namespace:  eventmodels.NamespaceID{ID: fixture.customer.Namespace},
		SubjectKey: fixture.subjectKey,
		StoredAt:   storedAt,
		RawEvents: []serializer.CloudEventsKafkaPayload{{
			Id:      raw.ID,
			Type:    raw.Type,
			Source:  raw.Source,
			Subject: raw.Subject,
			Time:    eventTime.Unix(),
			Data:    raw.Data,
		}},
	}
	msg, err := live.bus.Marshaler().Marshal(notification)
	require.NoError(t, err)

	return msg
}

func (live *liveProcess) logBalanceFailure(t *testing.T, client rte.Client, fixture liveBalanceFixture) {
	t.Helper()
	if !t.Failed() {
		return
	}

	lastCredit, creditErr := client.CreditBalances().Get(t.Context(), fixture.creditKey)
	lastMetered, meteredErr := client.MeteredEntitlements().Get(t.Context(), fixture.meteredKey)
	t.Logf("last credit: booked=%s live=%s err=%v; metered: balance=%g usage=%g err=%v",
		lastCredit.Value.Booked, lastCredit.Value.Live, creditErr, lastMetered.Value.Balance, lastMetered.Value.UsageInPeriod, meteredErr)

	for _, line := range strings.Split(live.metrics(t), "\n") {
		for _, prefix := range []string{"rte_upstream_events_total{", "rte_demand_operations_total{", "rte_calculations_total{", "rte_rating_lookups_total{"} {
			if strings.HasPrefix(line, prefix) {
				t.Log(line)
			}
		}
	}
}
