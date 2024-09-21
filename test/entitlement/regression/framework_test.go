// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package framework_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementrepo "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type Dependencies struct {
	DBClient  *db.Client
	PGDriver  *pgdriver.Driver
	EntDriver *entdriver.EntPostgresDriver

	GrantRepo           grant.Repo
	BalanceSnapshotRepo balance.SnapshotRepo
	GrantConnector      credit.GrantConnector

	EntitlementRepo entitlement.EntitlementRepo

	EntitlementConnector        entitlement.Connector
	StaticEntitlementConnector  staticentitlement.Connector
	BooleanEntitlementConnector booleanentitlement.Connector
	MeteredEntitlementConnector meteredentitlement.Connector

	Streaming *streamingtestutils.MockStreamingConnector

	FeatureRepo      productcatalog.FeatureRepo
	FeatureConnector productcatalog.FeatureConnector

	Log *slog.Logger
}

func (d *Dependencies) Close() {
	d.DBClient.Close()
	d.EntDriver.Close()
	d.PGDriver.Close()
}

func setupDependencies(t *testing.T) Dependencies {
	log := slog.Default()
	driver := testutils.InitPostgresDB(t)
	// init db
	dbClient := db.NewClient(db.Driver(driver.EntDriver.Driver()))
	if err := migrate.Up(driver.URL); err != nil {
		t.Fatalf("failed to migrate db: %s", err.Error())
	}

	// Init product catalog
	featureRepo := productcatalogrepo.NewPostgresFeatureRepo(dbClient, log)

	meters := []models.Meter{
		{
			Namespace:   "namespace-1",
			ID:          "meter-1",
			Slug:        "meter-1",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationCount,
		},
	}

	meterRepo := meter.NewInMemoryRepository(meters)

	featureConnector := productcatalog.NewFeatureConnector(featureRepo, meterRepo) // TODO: meter repo is needed

	// Init grants/credit
	grantRepo := grantrepo.NewPostgresGrantRepo(dbClient)
	balanceSnapshotRepo := grantrepo.NewPostgresBalanceSnapshotRepo(dbClient)

	// Init entitlements
	streaming := streamingtestutils.NewMockStreamingConnector(t)

	entitlementRepo := entitlementrepo.NewPostgresEntitlementRepo(dbClient)
	usageResetRepo := entitlementrepo.NewPostgresUsageResetRepo(dbClient)

	mockPublisher := eventbus.NewMock(t)

	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterRepo,
		log,
	)

	creditConnector := credit.NewCreditConnector(
		grantRepo,
		balanceSnapshotRepo,
		owner,
		streaming,
		log,
		time.Minute,
		mockPublisher,
	)

	meteredEntitlementConnector := meteredentitlement.NewMeteredEntitlementConnector(
		streaming,
		owner,
		creditConnector,
		creditConnector,
		grantRepo,
		entitlementRepo,
		mockPublisher,
	)

	staticEntitlementConnector := staticentitlement.NewStaticEntitlementConnector()
	booleanEntitlementConnector := booleanentitlement.NewBooleanEntitlementConnector()

	entitlementConnector := entitlement.NewEntitlementConnector(
		entitlementRepo,
		featureConnector,
		meterRepo,
		meteredEntitlementConnector,
		staticEntitlementConnector,
		booleanEntitlementConnector,
		mockPublisher,
	)

	return Dependencies{
		DBClient:  dbClient,
		PGDriver:  driver.PGDriver,
		EntDriver: driver.EntDriver,

		GrantRepo:      grantRepo,
		GrantConnector: creditConnector,

		EntitlementRepo: entitlementRepo,

		EntitlementConnector:        entitlementConnector,
		StaticEntitlementConnector:  staticEntitlementConnector,
		BooleanEntitlementConnector: booleanEntitlementConnector,
		MeteredEntitlementConnector: meteredEntitlementConnector,

		BalanceSnapshotRepo: balanceSnapshotRepo,

		Streaming: streaming,

		FeatureRepo:      featureRepo,
		FeatureConnector: featureConnector,

		Log: log,
	}
}
