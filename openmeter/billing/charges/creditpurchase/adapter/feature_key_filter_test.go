package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/lib/pq"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListChargesFeatureKeyFilter(t *testing.T) {
	suite.Run(t, new(ListChargesFeatureKeyFilterSuite))
}

type ListChargesFeatureKeyFilterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *db.Client

	taxCodeEnv *taxcodetestutils.TestEnv
	adapter    creditpurchase.Adapter

	namespace  string
	customerID string

	restrictedToA  string
	restrictedToAB string
	emptyFilters   string
	unrestricted   string
}

func (s *ListChargesFeatureKeyFilterSuite) SetupSuite() {
	t := s.T()

	s.testDB = testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	s.dbClient = db.NewClient(db.Driver(s.testDB.EntDriver.Driver()))

	s.taxCodeEnv = taxcodetestutils.NewTestEnvFromClient(t, s.dbClient, slog.Default())

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.dbClient,
		Logger: slog.Default(),
	})
	s.Require().NoError(err)

	s.adapter, err = New(Config{
		Client:      s.dbClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	s.Require().NoError(err)

	s.namespace = "test-list-charges-feature-key"

	customer, err := s.dbClient.Customer.Create().
		SetNamespace(s.namespace).
		SetName("test-customer").
		Save(t.Context())
	s.Require().NoError(err)
	s.customerID = customer.ID

	s.restrictedToA = s.insertCharge("restricted-to-a", pq.StringArray{"feature-a"})
	s.restrictedToAB = s.insertCharge("restricted-to-a-b", pq.StringArray{"feature-a", "feature-b"})
	s.emptyFilters = s.insertCharge("empty-filters", pq.StringArray{})
	s.unrestricted = s.insertCharge("unrestricted", nil)
}

func (s *ListChargesFeatureKeyFilterSuite) TearDownSuite() {
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *ListChargesFeatureKeyFilterSuite) insertCharge(name string, features pq.StringArray) string {
	s.T().Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)

	create := s.dbClient.ChargeCreditPurchase.Create().
		SetNamespace(s.namespace).
		SetCustomerID(s.customerID).
		SetServicePeriodFrom(now).
		SetServicePeriodTo(now.Add(time.Hour)).
		SetBillingPeriodFrom(now).
		SetBillingPeriodTo(now.Add(time.Hour)).
		SetFullServicePeriodFrom(now).
		SetFullServicePeriodTo(now.Add(time.Hour)).
		SetStatus(meta.ChargeStatusCreated).
		SetStatusDetailed(creditpurchase.StatusCreated).
		SetCurrency(currencyx.Code("USD")).
		SetManagedBy(billing.ManuallyManagedLine).
		SetName(name).
		SetTaxCodeID(s.taxCodeEnv.CreateTaxCode(s.T(), s.namespace).ID).
		SetCreditAmount(alpacadecimal.NewFromInt(100)).
		SetSettlement(creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}))
	if features != nil {
		create.SetFeatureFilters(features)
	}

	charge, err := create.Save(s.T().Context())
	s.Require().NoError(err)

	return charge.ID
}

func (s *ListChargesFeatureKeyFilterSuite) listIDs(featureKey *creditpurchase.FeatureKeyFilter) []string {
	s.T().Helper()

	result, err := s.adapter.ListCharges(s.T().Context(), creditpurchase.ListChargesInput{
		Page:        pagination.NewPage(1, 100),
		Namespace:   s.namespace,
		CustomerIDs: []string{s.customerID},
		FeatureKey:  featureKey,
	})
	s.Require().NoError(err)

	return lo.Map(result.Items, func(item creditpurchase.Charge, _ int) string {
		return item.ID
	})
}

func (s *ListChargesFeatureKeyFilterSuite) TestKeyedFilterMatchesByOverlap() {
	s.Require().ElementsMatch(
		[]string{s.restrictedToA, s.restrictedToAB},
		s.listIDs(&creditpurchase.FeatureKeyFilter{In: []string{"feature-a"}}),
		"a keyed filter matches every grant whose restriction includes the key; unrestricted grants stay out",
	)

	s.Require().ElementsMatch(
		[]string{s.restrictedToAB},
		s.listIDs(&creditpurchase.FeatureKeyFilter{In: []string{"feature-b"}}),
		"a multi-feature grant matches on any one of its features",
	)

	s.Require().ElementsMatch(
		[]string{s.restrictedToAB},
		s.listIDs(&creditpurchase.FeatureKeyFilter{In: []string{"feature-b", "feature-c"}}),
		"multiple keys match with any-of semantics",
	)

	s.Require().Empty(
		s.listIDs(&creditpurchase.FeatureKeyFilter{In: []string{"feature-c"}}),
		"an unknown key matches nothing",
	)
}

func (s *ListChargesFeatureKeyFilterSuite) TestExistsFilter() {
	s.Require().ElementsMatch(
		[]string{s.emptyFilters, s.unrestricted},
		s.listIDs(&creditpurchase.FeatureKeyFilter{Exists: lo.ToPtr(false)}),
		"exists=false selects unrestricted grants, whether stored as NULL or as an empty array",
	)

	s.Require().ElementsMatch(
		[]string{s.restrictedToA, s.restrictedToAB},
		s.listIDs(&creditpurchase.FeatureKeyFilter{Exists: lo.ToPtr(true)}),
		"exists=true selects only feature-restricted grants",
	)
}

func (s *ListChargesFeatureKeyFilterSuite) TestNilFilterReturnsEverything() {
	s.Require().ElementsMatch(
		[]string{s.restrictedToA, s.restrictedToAB, s.emptyFilters, s.unrestricted},
		s.listIDs(nil),
	)
}
