package meteredentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ResetEntitlementUsageParams struct {
	At           time.Time
	RetainAnchor bool
}

type Connector interface {
	entitlement.SubTypeConnector

	GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*EntitlementBalance, error)
	GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, credit.GrantBurnDownHistory, error)
	ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, params ResetEntitlementUsageParams) (balanceAfterReset *EntitlementBalance, err error)

	ResetEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespace string, highwatermark time.Time) ([]models.NamespacedID, error)

	// GetEntitlementGrantBalanceHistory(ctx context.Context, entitlementGrantID EntitlementGrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
	CreateGrant(ctx context.Context, namespace string, subjectKey string, entitlementIdOrFeatureKey string, inputGrant CreateEntitlementGrantInputs) (EntitlementGrant, error)
	ListEntitlementGrants(ctx context.Context, namespace string, subjectKey string, entitlementIdOrFeatureKey string) ([]EntitlementGrant, error)
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

func (c *connector) BeforeCreate(model entitlement.CreateEntitlementInputs, feature productcatalog.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
	model.EntitlementType = entitlement.EntitlementTypeMetered

	if model.Config != nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "Config is not allowed for metered entitlements"}
	}

	if model.UsagePeriod == nil {
		return nil, &entitlement.InvalidValueError{Message: "UsagePeriod is required for metered entitlements", Type: entitlement.EntitlementTypeMetered}
	}

	if feature.MeterSlug == nil {
		return nil, &entitlement.InvalidFeatureError{FeatureID: feature.ID, Message: "Feature has no meter"}
	}

	if model.IssueAfterResetPriority != nil && model.IssueAfterReset == nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: "IssueAfterResetPriority requires IssueAfterReset"}
	}

	model.MeasureUsageFrom = convert.ToPointer(defaultx.WithDefault(model.MeasureUsageFrom, clock.Now().Truncate(c.granularity)))
	model.IsSoftLimit = convert.ToPointer(defaultx.WithDefault(model.IsSoftLimit, false))
	model.IssueAfterReset = convert.ToPointer(defaultx.WithDefault(model.IssueAfterReset, 0.0))

	model.UsagePeriod.Anchor = model.UsagePeriod.Anchor.Truncate(c.granularity)

	// Calculating the very first period is different as it has to start from the start of measurement
	currentPeriod, err := model.UsagePeriod.GetCurrentPeriodAt(*model.MeasureUsageFrom)
	if err != nil {
		return nil, err
	}

	if model.MeasureUsageFrom.After(currentPeriod.To) || model.MeasureUsageFrom.Equal(currentPeriod.To) {
		return nil, fmt.Errorf("inconsistency error: start of measurement %s is after or equal to the calculated period end %s, period end should be exclusive", model.MeasureUsageFrom, currentPeriod)
	}

	// We have to alter the period to start with start of measurement
	currentPeriod.From = *model.MeasureUsageFrom

	return &entitlement.CreateEntitlementRepoInputs{
		Namespace:               model.Namespace,
		FeatureID:               feature.ID,
		FeatureKey:              feature.Key,
		SubjectKey:              model.SubjectKey,
		EntitlementType:         model.EntitlementType,
		Metadata:                model.Metadata,
		MeasureUsageFrom:        model.MeasureUsageFrom,
		IssueAfterReset:         model.IssueAfterReset,
		IssueAfterResetPriority: model.IssueAfterResetPriority,
		IsSoftLimit:             model.IsSoftLimit,
		UsagePeriod:             model.UsagePeriod,
		CurrentUsagePeriod:      &currentPeriod,
	}, nil
}

func (c *connector) AfterCreate(ctx context.Context, end *entitlement.Entitlement) error {
	metered, err := ParseFromGenericEntitlement(end)
	if err != nil {
		return err
	}

	// Right now transaction is magically passed through ctx here.
	// Until we refactor and fix this, to avoid any potential errors due to changes in downstream connectors, the code is inlined here.
	// issue default grants
	if metered.HasDefaultGrant() {
		if metered.IssueAfterReset == nil {
			return fmt.Errorf("inconsistency error: entitlement %s should have default grant but has no IssueAfterReset", metered.ID)
		}

		effectiveAt := metered.CurrentUsagePeriod.From
		amountToIssue := metered.IssueAfterReset.Amount
		_, err := c.grantConnector.CreateGrant(ctx, credit.NamespacedGrantOwner{
			Namespace: metered.Namespace,
			ID:        credit.GrantOwner(metered.ID),
		}, credit.CreateGrantInput{
			Amount:      amountToIssue,
			Priority:    defaultx.WithDefault(metered.IssueAfterReset.Priority, DefaultIssueAfterResetPriority),
			EffectiveAt: effectiveAt,
			Expiration: credit.ExpirationPeriod{
				Count:    100, // This is a bit of an issue... It would make sense for recurring tags to not have an expiration
				Duration: credit.ExpirationPeriodDurationYear,
			},
			// These two in conjunction make the grant always have `amountToIssue` balance after a reset
			ResetMaxRollover: amountToIssue,
			ResetMinRollover: amountToIssue,
			Metadata: map[string]string{
				IssueAfterResetMetaTag: "true",
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
