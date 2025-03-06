package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client *db.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("postgres client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func New(config Config) (meter.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

var _ meter.Service = (*adapter)(nil)

type adapter struct {
	db     *db.Client
	logger *slog.Logger
}

type ManageConfig struct {
	Config

	EntitlementRepository entitlement.EntitlementRepo
	FeatureRepository     feature.FeatureRepo
	NamespaceManager      *namespace.Manager
	StreamingConnector    streaming.Connector
}

func (c ManageConfig) Validate() error {
	if err := c.Config.Validate(); err != nil {
		return err
	}

	if c.EntitlementRepository == nil {
		return errors.New("entitlement repository is required")
	}

	if c.FeatureRepository == nil {
		return errors.New("feature repository is required")
	}

	if c.StreamingConnector == nil {
		return errors.New("streaming connector is required")
	}

	return nil
}

func NewManage(config ManageConfig) (meter.ManageService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	service, err := New(config.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	return &manageAdapter{
		Service: service,
		db:      config.Client,
		logger:  config.Logger,

		entitlementRepository: config.EntitlementRepository,
		featureRepository:     config.FeatureRepository,
		namespaceManager:      config.NamespaceManager,
		streamingConnector:    config.StreamingConnector,
	}, nil
}

var _ meter.ManageService = (*manageAdapter)(nil)

type manageAdapter struct {
	meter.Service

	db     *db.Client
	logger *slog.Logger

	entitlementRepository entitlement.EntitlementRepo
	featureRepository     feature.FeatureRepo
	namespaceManager      *namespace.Manager
	streamingConnector    streaming.Connector
}

// Tx implements entutils.TxCreator interface
func (a manageAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a manageAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *manageAdapter {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &manageAdapter{
		db:     txClient.Client(),
		logger: a.logger,

		entitlementRepository: a.entitlementRepository,
		featureRepository:     a.featureRepository,
		namespaceManager:      a.namespaceManager,
		streamingConnector:    a.streamingConnector,
	}
}
