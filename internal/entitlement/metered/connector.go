package meteredentitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Connector interface {
	entitlement.SubTypeConnector

	GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*EntitlementBalance, error)
	GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, credit.GrantBurnDownHistory, error)
	ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, resetAt time.Time) (balanceAfterReset *EntitlementBalance, err error)

	// GetEntitlementGrantBalanceHistory(ctx context.Context, entitlementGrantID EntitlementGrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
	CreateGrant(ctx context.Context, entitlement models.NamespacedID, inputGrant CreateEntitlementGrantInputs) (EntitlementGrant, error)
	ListEntitlementGrants(ctx context.Context, entitlementID models.NamespacedID) ([]EntitlementGrant, error)
}

type MeteredEntitlementValue struct {
	isSoftLimit   bool      `json:"-"`
	Balance       float64   `json:"balance"`
	UsageInPeriod float64   `json:"usageInPeriod"`
	Overage       float64   `json:"overage"`
	StartOfPeriod time.Time `json:"startOfPeriod"`
}

var _ entitlement.EntitlementValue = &MeteredEntitlementValue{}

func (m *MeteredEntitlementValue) HasAccess() bool {
	if m.isSoftLimit {
		return true
	}
	return m.Balance > 0
}

type connector struct {
	streamingConnector streaming.Connector
	ownerConnector     credit.OwnerConnector
	balanceConnector   credit.BalanceConnector
	grantConnector     credit.GrantConnector
	entitlementRepo    entitlement.EntitlementRepo

	granularity time.Duration
}

func NewMeteredEntitlementConnector(
	streamingConnector streaming.Connector,
	ownerConnector credit.OwnerConnector,
	balanceConnector credit.BalanceConnector,
	grantConnector credit.GrantConnector,
	entitlementRepo entitlement.EntitlementRepo,
) Connector {
	return &connector{
		streamingConnector: streamingConnector,
		ownerConnector:     ownerConnector,
		balanceConnector:   balanceConnector,
		grantConnector:     grantConnector,
		entitlementRepo:    entitlementRepo,

		// FIXME: This should be configurable
		granularity: time.Minute,
	}
}

func (e *connector) GetValue(entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	metered, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	balance, err := e.GetEntitlementBalance(context.Background(), models.NamespacedID{
		Namespace: metered.Namespace,
		ID:        metered.ID,
	}, at)
	if err != nil {
		return nil, err
	}

	return &MeteredEntitlementValue{
		isSoftLimit:   metered.IsSoftLimit,
		Balance:       balance.Balance,
		UsageInPeriod: balance.UsageInPeriod,
		Overage:       balance.Overage,
		StartOfPeriod: balance.StartOfPeriod,
	}, nil
}

func (c *connector) SetDefaultsAndValidate(model *entitlement.CreateEntitlementInputs) error {
	model.EntitlementType = entitlement.EntitlementTypeMetered
	model.MeasureUsageFrom = convert.ToPointer(defaultx.WithDefault(model.MeasureUsageFrom, time.Now().Truncate(c.granularity)))
	model.IsSoftLimit = convert.ToPointer(defaultx.WithDefault(model.IsSoftLimit, false))
	model.IssueAfterReset = convert.ToPointer(defaultx.WithDefault(model.IssueAfterReset, 0.0))

	if model.Config != nil {
		return &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is not allowed for metered entitlements"}
	}

	if model.UsagePeriod == nil {
		return &entitlement.InvalidValueError{Message: "UsagePeriod is required for metered entitlements", Type: entitlement.EntitlementTypeMetered}
	}

	return nil
}

func (c *connector) ValidateForFeature(model *entitlement.CreateEntitlementInputs, feature productcatalog.Feature) error {
	if feature.MeterSlug == nil {
		return &entitlement.InvalidFeatureError{FeatureID: feature.ID, Message: "Feature has no meter"}
	}
	return nil
}
