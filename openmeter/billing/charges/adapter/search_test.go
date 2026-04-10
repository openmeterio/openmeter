package adapter

import (
	"context"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestListCustomersToAdvance(t *testing.T) {
	suite.Run(t, new(ListCustomersToAdvanceSuite))
}

type ListCustomersToAdvanceSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *db.Client
	adapter  charges.ChargesSearchAdapter
}

func (s *ListCustomersToAdvanceSuite) SetupSuite() {
	t := s.T()

	s.testDB = testutils.InitPostgresDB(t)
	s.dbClient = db.NewClient(db.Driver(s.testDB.EntDriver.Driver()))

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: s.testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           slog.Default(),
	})
	require.NoError(t, err)
	defer migrator.CloseOrLogError()
	require.NoError(t, migrator.Up())

	a, err := New(Config{
		Client: s.dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)
	s.adapter = a
}

func (s *ListCustomersToAdvanceSuite) TearDownSuite() {
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

// createCustomer creates a customer record and returns its generated ID.
func (s *ListCustomersToAdvanceSuite) createCustomer(namespace string) string {
	s.T().Helper()

	c, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(context.Background())
	s.Require().NoError(err)

	return c.ID
}

// insertFlatFeeCharge inserts a minimal flat fee charge row for testing the search view.
func (s *ListCustomersToAdvanceSuite) insertFlatFeeCharge(namespace, customerID string, status meta.ChargeStatus, advanceAfter *time.Time) {
	s.T().Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)

	create := s.dbClient.ChargeFlatFee.Create().
		SetNamespace(namespace).
		SetCustomerID(customerID).
		SetStatus(status).
		SetStatusDetailed(flatfee.Status(status)).
		SetCurrency(currencyx.Code("USD")).
		SetManagedBy(billing.SubscriptionManagedLine).
		SetName("test-charge").
		SetPaymentTerm(productcatalog.InArrearsPaymentTerm).
		SetInvoiceAt(now).
		SetSettlementMode(productcatalog.CreditOnlySettlementMode).
		SetProRating(flatfee.NoProratingAdapterMode).
		SetAmountBeforeProration(alpacadecimal.NewFromInt(100)).
		SetAmountAfterProration(alpacadecimal.NewFromInt(100)).
		SetServicePeriodFrom(now).
		SetServicePeriodTo(now.Add(time.Hour)).
		SetBillingPeriodFrom(now).
		SetBillingPeriodTo(now.Add(time.Hour)).
		SetFullServicePeriodFrom(now).
		SetFullServicePeriodTo(now.Add(time.Hour))

	if advanceAfter != nil {
		create = create.SetAdvanceAfter(*advanceAfter)
	}

	_, err := create.Save(context.Background())
	s.Require().NoError(err)
}

func (s *ListCustomersToAdvanceSuite) TestReturnsOnlyEligibleCustomers() {
	ctx := context.Background()
	ns := "test-eligible"
	now := time.Now().UTC().Truncate(time.Microsecond)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	eligibleID := s.createCustomer(ns)
	futureID := s.createCustomer(ns)
	finalID := s.createCustomer(ns)
	deletedID := s.createCustomer(ns)
	nilID := s.createCustomer(ns)

	s.insertFlatFeeCharge(ns, eligibleID, meta.ChargeStatusActive, &past)
	s.insertFlatFeeCharge(ns, futureID, meta.ChargeStatusActive, &future)
	s.insertFlatFeeCharge(ns, finalID, meta.ChargeStatusFinal, &past)
	s.insertFlatFeeCharge(ns, deletedID, meta.ChargeStatusDeleted, &past)
	s.insertFlatFeeCharge(ns, nilID, meta.ChargeStatusActive, nil)

	result, err := s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Namespaces:      []string{ns},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)

	s.Require().Len(result.Items, 1)
	s.Equal(customer.CustomerID{Namespace: ns, ID: eligibleID}, result.Items[0])
}

func (s *ListCustomersToAdvanceSuite) TestDeduplicatesCustomers() {
	ctx := context.Background()
	ns := "test-dedup"
	past := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().UTC().Truncate(time.Microsecond)

	custID := s.createCustomer(ns)

	// Same customer with two charges
	s.insertFlatFeeCharge(ns, custID, meta.ChargeStatusActive, &past)
	s.insertFlatFeeCharge(ns, custID, meta.ChargeStatusActive, &past)

	result, err := s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Namespaces:      []string{ns},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)

	s.Require().Len(result.Items, 1)
	s.Equal(custID, result.Items[0].ID)
}

func (s *ListCustomersToAdvanceSuite) TestStableOrdering() {
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().UTC().Truncate(time.Microsecond)

	nsA := "test-order-a"
	nsB := "test-order-b"

	custA1 := s.createCustomer(nsA)
	custA2 := s.createCustomer(nsA)
	custB1 := s.createCustomer(nsB)
	custB2 := s.createCustomer(nsB)

	// Insert in deliberately non-sorted order
	s.insertFlatFeeCharge(nsB, custB2, meta.ChargeStatusActive, &past)
	s.insertFlatFeeCharge(nsA, custA1, meta.ChargeStatusActive, &past)
	s.insertFlatFeeCharge(nsB, custB1, meta.ChargeStatusActive, &past)
	s.insertFlatFeeCharge(nsA, custA2, meta.ChargeStatusActive, &past)

	result, err := s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Namespaces:      []string{nsA, nsB},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 4)

	// Build expected order: sorted by (namespace, customer_id)
	expected := []customer.CustomerID{
		{Namespace: nsA, ID: custA1},
		{Namespace: nsA, ID: custA2},
		{Namespace: nsB, ID: custB1},
		{Namespace: nsB, ID: custB2},
	}
	sort.Slice(expected, func(i, j int) bool {
		if expected[i].Namespace != expected[j].Namespace {
			return expected[i].Namespace < expected[j].Namespace
		}
		return expected[i].ID < expected[j].ID
	})

	s.Equal(expected, result.Items)
}

func (s *ListCustomersToAdvanceSuite) TestPagination() {
	ctx := context.Background()
	ns := "test-pagination"
	past := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Create 5 customers and collect their sorted IDs
	var custIDs []string
	for i := 0; i < 5; i++ {
		custIDs = append(custIDs, s.createCustomer(ns))
	}
	sort.Strings(custIDs)

	for _, id := range custIDs {
		s.insertFlatFeeCharge(ns, id, meta.ChargeStatusActive, &past)
	}

	// Page 1: size 2
	result, err := s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Page:            pagination.Page{PageSize: 2, PageNumber: 1},
		Namespaces:      []string{ns},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 2)
	s.Equal(custIDs[0], result.Items[0].ID)
	s.Equal(custIDs[1], result.Items[1].ID)

	// Page 2: size 2
	result, err = s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Page:            pagination.Page{PageSize: 2, PageNumber: 2},
		Namespaces:      []string{ns},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 2)
	s.Equal(custIDs[2], result.Items[0].ID)
	s.Equal(custIDs[3], result.Items[1].ID)

	// Page 3: size 2 - last page with 1 item
	result, err = s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Page:            pagination.Page{PageSize: 2, PageNumber: 3},
		Namespaces:      []string{ns},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 1)
	s.Equal(custIDs[4], result.Items[0].ID)
}

func (s *ListCustomersToAdvanceSuite) TestNamespaceFilter() {
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().UTC().Truncate(time.Microsecond)

	includeID := s.createCustomer("ns-include")
	s.createCustomer("ns-exclude")
	excludeID := s.createCustomer("ns-exclude")

	s.insertFlatFeeCharge("ns-include", includeID, meta.ChargeStatusActive, &past)
	s.insertFlatFeeCharge("ns-exclude", excludeID, meta.ChargeStatusActive, &past)

	result, err := s.adapter.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{
		Namespaces:      []string{"ns-include"},
		AdvanceAfterLTE: now,
	})
	s.Require().NoError(err)

	s.Require().Len(result.Items, 1)
	s.Equal("ns-include", result.Items[0].Namespace)
	s.Equal(includeID, result.Items[0].ID)
}
