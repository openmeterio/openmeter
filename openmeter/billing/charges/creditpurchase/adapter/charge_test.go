package adapter

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCreateChargeKeyConflict(t *testing.T) {
	suite.Run(t, new(CreateChargeKeyConflictSuite))
}

type CreateChargeKeyConflictSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *db.Client

	taxCodeEnv *taxcodetestutils.TestEnv
	adapter    creditpurchase.Adapter
}

func (s *CreateChargeKeyConflictSuite) SetupSuite() {
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
}

func (s *CreateChargeKeyConflictSuite) TearDownSuite() {
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *CreateChargeKeyConflictSuite) createCustomer(namespace string) string {
	s.T().Helper()

	c, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(s.T().Context())
	s.Require().NoError(err)

	return c.ID
}

func (s *CreateChargeKeyConflictSuite) newCreateInput(namespace, customerID string, key *string) creditpurchase.CreateChargeInput {
	s.T().Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)
	period := timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)}

	return creditpurchase.CreateChargeInput{
		Namespace: namespace,
		Intent: creditpurchase.Intent{
			Intent: meta.Intent{
				ManagedBy:  billing.ManuallyManagedLine,
				CustomerID: customerID,
				Currency:   currencyx.Code("USD"),
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: s.taxCodeEnv.CreateTaxCode(s.T(), namespace).ID,
				},
			},
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "key conflict grant",
					ServicePeriod:     period,
					BillingPeriod:     period,
					FullServicePeriod: period,
				},
				CreditAmount: alpacadecimal.NewFromInt(10),
				Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
			},
			Key: key,
		},
	}
}

func (s *CreateChargeKeyConflictSuite) TestDuplicateKeyReturnsKeyConflictError() {
	ctx := s.T().Context()
	ns := "test-create-charge-key-conflict"
	customerID := s.createCustomer(ns)
	key := "conflict-key-1"

	_, err := s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, &key))
	s.Require().NoError(err)

	_, err = s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, &key))
	s.Require().Error(err)

	var conflict *creditpurchase.ChargeKeyConflictError
	s.Require().True(errors.As(err, &conflict), "expected ChargeKeyConflictError, got: %v", err)
	s.Require().Equal(ns, conflict.Namespace)
	s.Require().Equal(customerID, conflict.CustomerID)
	s.Require().Equal(key, conflict.Key)
	s.Require().True(models.IsGenericConflictError(err))
	s.Require().NotContains(err.Error(), "duplicate key value", "raw driver error must not leak into the message")
}

func (s *CreateChargeKeyConflictSuite) TestSoftDeletedChargeReleasesKey() {
	ctx := s.T().Context()
	ns := "test-create-charge-key-release"
	customerID := s.createCustomer(ns)
	key := "reusable-key-1"

	first, err := s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, &key))
	s.Require().NoError(err)

	_, err = s.dbClient.ChargeCreditPurchase.UpdateOneID(first.ID).
		SetDeletedAt(time.Now().UTC()).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, &key))
	s.Require().NoError(err, "a soft-deleted charge must not reserve the key")
}

func (s *CreateChargeKeyConflictSuite) TestDistinctAndAbsentKeysDoNotConflict() {
	ctx := s.T().Context()
	ns := "test-create-charge-key-distinct"
	customerID := s.createCustomer(ns)

	_, err := s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, lo.ToPtr("key-a")))
	s.Require().NoError(err)

	_, err = s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, lo.ToPtr("key-b")))
	s.Require().NoError(err)

	_, err = s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, nil))
	s.Require().NoError(err)

	_, err = s.adapter.CreateCharge(ctx, s.newCreateInput(ns, customerID, nil))
	s.Require().NoError(err, "absent keys must not conflict with each other")
}
