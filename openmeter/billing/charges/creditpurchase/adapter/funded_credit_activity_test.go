package adapter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestListFundedCreditActivities(t *testing.T) {
	suite.Run(t, new(ListFundedCreditActivitiesSuite))
}

type ListFundedCreditActivitiesSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *db.Client
}

func (s *ListFundedCreditActivitiesSuite) SetupSuite() {
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
}

func (s *ListFundedCreditActivitiesSuite) TearDownSuite() {
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *ListFundedCreditActivitiesSuite) createCustomer(namespace string) string {
	s.T().Helper()

	c, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(context.Background())
	s.Require().NoError(err)

	return c.ID
}

func (s *ListFundedCreditActivitiesSuite) insertCreditPurchaseWithGrant(
	namespace string,
	customerID string,
	currency currencyx.Code,
	chargeCreatedAt time.Time,
	fundedAt time.Time,
	name string,
	description *string,
) meta.ChargeID {
	s.T().Helper()

	servicePeriodTo := chargeCreatedAt.Add(time.Hour)

	chargeEntity, err := s.dbClient.ChargeCreditPurchase.Create().
		SetNamespace(namespace).
		SetCustomerID(customerID).
		SetServicePeriodFrom(chargeCreatedAt).
		SetServicePeriodTo(servicePeriodTo).
		SetBillingPeriodFrom(chargeCreatedAt).
		SetBillingPeriodTo(servicePeriodTo).
		SetFullServicePeriodFrom(chargeCreatedAt).
		SetFullServicePeriodTo(servicePeriodTo).
		SetStatus(meta.ChargeStatusCreated).
		SetStatusDetailed(creditpurchase.StatusCreated).
		SetCurrency(currency).
		SetManagedBy(billing.SubscriptionManagedLine).
		SetName(name).
		SetNillableDescription(description).
		SetCreditAmount(alpacadecimal.NewFromInt(100)).
		SetSettlement(creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{})).
		SetCreatedAt(chargeCreatedAt).
		SetUpdatedAt(chargeCreatedAt).
		Save(s.T().Context())
	s.Require().NoError(err)

	_, err = s.dbClient.ChargeCreditPurchaseCreditGrant.Create().
		SetNamespace(namespace).
		SetChargeID(chargeEntity.ID).
		SetTransactionGroupID(ulid.Make().String()).
		SetGrantedAt(fundedAt).
		SetCreditPurchaseID(chargeEntity.ID).
		SetCreatedAt(fundedAt).
		SetUpdatedAt(fundedAt).
		Save(s.T().Context())
	s.Require().NoError(err)

	return meta.ChargeID{
		Namespace: namespace,
		ID:        chargeEntity.ID,
	}
}

func (s *ListFundedCreditActivitiesSuite) TestPaginatesWithAfter() {
	ctx := context.Background()
	ns := "test-funded-activity-cursors"
	customerID := s.createCustomer(ns)
	base := time.Now().UTC().Truncate(time.Microsecond)

	idNewest := s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(1*time.Minute),
		base.Add(3*time.Minute),
		"newest-funded",
		nil,
	)
	idMiddle := s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(3*time.Minute),
		base.Add(2*time.Minute),
		"middle-funded",
		nil,
	)
	idOldest := s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(2*time.Minute),
		base.Add(2*time.Minute),
		"oldest-funded",
		nil,
	)

	customerRef := customer.CustomerID{Namespace: ns, ID: customerID}

	page1, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customerRef,
		Limit:    2,
	})
	s.Require().NoError(err)
	s.Require().Len(page1.Items, 2)
	s.False(page1.HasPrevious)
	s.NotNil(page1.NextCursor)
	s.Equal(idNewest, page1.Items[0].ChargeID)
	s.Equal(idMiddle, page1.Items[1].ChargeID)

	page2, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customerRef,
		Limit:    2,
		After:    page1.NextCursor,
	})
	s.Require().NoError(err)
	s.Require().Len(page2.Items, 1)
	s.True(page2.HasPrevious)
	s.Nil(page2.NextCursor)
	s.Equal(idOldest, page2.Items[0].ChargeID)
}

func (s *ListFundedCreditActivitiesSuite) TestPaginatesWithBefore() {
	ctx := context.Background()
	ns := "test-funded-activity-before"
	customerID := s.createCustomer(ns)
	base := time.Now().UTC().Truncate(time.Microsecond)

	s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(30*time.Second),
		base.Add(5*time.Minute),
		"funded-5",
		nil,
	)
	s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(1*time.Minute),
		base.Add(4*time.Minute),
		"funded-4",
		nil,
	)
	s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(2*time.Minute),
		base.Add(3*time.Minute),
		"funded-3",
		nil,
	)
	s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(3*time.Minute),
		base.Add(2*time.Minute),
		"funded-2",
		nil,
	)
	s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(4*time.Minute),
		base.Add(1*time.Minute),
		"funded-1",
		nil,
	)

	customerRef := customer.CustomerID{Namespace: ns, ID: customerID}

	initialPage, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customerRef,
		Limit:    2,
	})
	s.Require().NoError(err)
	s.Require().NotNil(initialPage.NextCursor)
	s.Require().Len(initialPage.Items, 2)
	s.Equal("funded-5", initialPage.Items[0].Name)
	s.Equal("funded-4", initialPage.Items[1].Name)

	page2, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customerRef,
		Limit:    2,
		After:    initialPage.NextCursor,
	})
	s.Require().NoError(err)
	s.Require().Len(page2.Items, 2)
	s.Equal("funded-3", page2.Items[0].Name)
	s.Equal("funded-2", page2.Items[1].Name)

	page1, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customerRef,
		Limit:    2,
		Before: &creditpurchase.FundedCreditActivityCursor{
			FundedAt:        page2.Items[1].FundedAt,
			ChargeCreatedAt: page2.Items[1].ChargeCreatedAt,
			ChargeID:        page2.Items[1].ChargeID,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(page1.Items, 2)
	s.Require().NotNil(page1.NextCursor)
	s.Equal("funded-4", page1.Items[0].Name)
	s.Equal("funded-3", page1.Items[1].Name)

	pageForward, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customerRef,
		Limit:    2,
		After:    page1.NextCursor,
	})
	s.Require().NoError(err)
	s.Require().Len(pageForward.Items, 2)
	s.Equal("funded-2", pageForward.Items[0].Name)
	s.Equal("funded-1", pageForward.Items[1].Name)
}

func (s *ListFundedCreditActivitiesSuite) TestFiltersByCurrency() {
	ctx := context.Background()
	ns := "test-funded-activity-currency"
	customerID := s.createCustomer(ns)
	base := time.Now().UTC().Truncate(time.Microsecond)

	idUSD := s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("USD"),
		base.Add(1*time.Minute),
		base.Add(2*time.Minute),
		"usd-funded",
		nil,
	)
	s.insertCreditPurchaseWithGrant(
		ns,
		customerID,
		currencyx.Code("EUR"),
		base.Add(2*time.Minute),
		base.Add(3*time.Minute),
		"eur-funded",
		nil,
	)

	usd := currencyx.Code("USD")
	result, err := ListFundedCreditActivities(ctx, s.dbClient, creditpurchase.ListFundedCreditActivitiesInput{
		Customer: customer.CustomerID{Namespace: ns, ID: customerID},
		Limit:    10,
		Currency: &usd,
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 1)
	s.Equal(idUSD, result.Items[0].ChargeID)
	s.Equal(usd, result.Items[0].Currency)
}
