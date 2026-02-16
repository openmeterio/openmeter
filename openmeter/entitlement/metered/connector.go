package meteredentitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ResetEntitlementUsageParams struct {
	At              time.Time
	RetainAnchor    bool
	PreserveOverage *bool
}

type Connector interface {
	models.ServiceHooks[Entitlement]
	entitlement.SubTypeConnector

	GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*EntitlementBalance, error)
	GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, engine.GrantBurnDownHistory, error)
	ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, params ResetEntitlementUsageParams) (balanceAfterReset *EntitlementBalance, err error)

	ResetEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespace string, highwatermark time.Time) ([]models.NamespacedID, error)

	// GetEntitlementGrantBalanceHistory(ctx context.Context, entitlementGrantID EntitlementGrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
	CreateGrant(ctx context.Context, namespace string, customerID string, entitlementIdOrFeatureKey string, inputGrant CreateEntitlementGrantInputs) (EntitlementGrant, error)
	ListEntitlementGrants(ctx context.Context, namespace string, params ListEntitlementGrantsParams) (pagination.Result[EntitlementGrant], error)
}

type MeteredEntitlementValue struct {
	isSoftLimit               bool      `json:"-"`
	Balance                   float64   `json:"balance"`
	UsageInPeriod             float64   `json:"usageInPeriod"`
	Overage                   float64   `json:"overage"`
	TotalAvailableGrantAmount float64   `json:"totalAvailableGrantAmount"`
	StartOfPeriod             time.Time `json:"startOfPeriod"`
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
	ownerConnector     grant.OwnerConnector
	balanceConnector   credit.BalanceConnector
	grantConnector     credit.GrantConnector
	grantRepo          grant.Repo
	entitlementRepo    entitlement.EntitlementRepo

	granularity time.Duration
	publisher   eventbus.Publisher
	hooks       models.ServiceHookRegistry[Entitlement]

	logger *slog.Logger
	tracer trace.Tracer
}

func NewMeteredEntitlementConnector(
	streamingConnector streaming.Connector,
	ownerConnector grant.OwnerConnector,
	balanceConnector credit.BalanceConnector,
	grantConnector credit.GrantConnector,
	grantRepo grant.Repo,
	entitlementRepo entitlement.EntitlementRepo,
	publisher eventbus.Publisher,
	logger *slog.Logger,
	tracer trace.Tracer,
) Connector {
	return &connector{
		streamingConnector: streamingConnector,
		ownerConnector:     ownerConnector,
		balanceConnector:   balanceConnector,
		grantConnector:     grantConnector,
		grantRepo:          grantRepo,
		entitlementRepo:    entitlementRepo,

		// FIXME: This should be configurable
		granularity: time.Minute,

		publisher: publisher,
		logger:    logger,
		tracer:    tracer,
		hooks:     models.ServiceHookRegistry[Entitlement]{},
	}
}

func (c *connector) RegisterHooks(hooks ...models.ServiceHook[Entitlement]) {
	c.hooks.RegisterHooks(hooks...)
}

func (e *connector) GetValue(ctx context.Context, entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	e.logger.DebugContext(ctx, "Getting entitlement value", "entitlement", entitlement, "at", at)

	metered, err := ParseFromGenericEntitlement(entitlement)
	if err != nil {
		return nil, err
	}

	balance, err := e.GetEntitlementBalance(ctx, models.NamespacedID{
		Namespace: metered.Namespace,
		ID:        metered.ID,
	}, at)
	if err != nil {
		return nil, err
	}

	return &MeteredEntitlementValue{
		isSoftLimit:               metered.IsSoftLimit,
		Balance:                   balance.Balance,
		UsageInPeriod:             balance.UsageInPeriod,
		Overage:                   balance.Overage,
		StartOfPeriod:             balance.StartOfPeriod,
		TotalAvailableGrantAmount: balance.TotalAvailableGrantAmount,
	}, nil
}

func (c *connector) BeforeCreate(model entitlement.CreateEntitlementInputs, feature feature.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
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

	measureUsageFrom := convert.SafeDeRef(
		model.MeasureUsageFrom,
		func(m entitlement.MeasureUsageFromInput) *time.Time {
			return convert.ToPointer(m.Get())
		},
	)

	measureUsageFrom = convert.ToPointer(defaultx.WithDefault(measureUsageFrom, clock.Now()).Truncate(c.granularity))

	model.IsSoftLimit = convert.ToPointer(defaultx.WithDefault(model.IsSoftLimit, false))
	model.IssueAfterReset = convert.ToPointer(defaultx.WithDefault(model.IssueAfterReset, 0.0))

	// Lets truncate the anchor
	truncated := model.UsagePeriod.GetValue().Anchor.Truncate(c.granularity)
	model.UsagePeriod = lo.ToPtr(timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
		return *measureUsageFrom
		// return truncated
	})(timeutil.Recurrence{
		Interval: model.UsagePeriod.GetValue().Interval,
		Anchor:   truncated,
	}))

	// Let's validate the usage period isn't less than 1h
	if err := model.UsagePeriod.GetValue().Validate(); err != nil {
		return nil, &entitlement.InvalidValueError{Type: model.EntitlementType, Message: err.Error()}
	}

	// Calculating the very first period is different as it has to start from the start of measurement
	currentPeriod, err := model.UsagePeriod.GetValue().GetPeriodAt(*measureUsageFrom) // FIXME: this might be incorrect
	if err != nil {
		return nil, err
	}

	if measureUsageFrom.After(currentPeriod.To) || measureUsageFrom.Equal(currentPeriod.To) {
		return nil, fmt.Errorf("inconsistency error: start of measurement %s is after or equal to the calculated period end %s, period end should be exclusive", measureUsageFrom, currentPeriod)
	}

	// We have to alter the period to start with start of measurement
	currentPeriod.From = *measureUsageFrom

	return &entitlement.CreateEntitlementRepoInputs{
		Namespace:               model.Namespace,
		FeatureID:               feature.ID,
		FeatureKey:              feature.Key,
		UsageAttribution:        model.UsageAttribution,
		EntitlementType:         model.EntitlementType,
		Metadata:                model.Metadata,
		Annotations:             model.Annotations,
		MeasureUsageFrom:        measureUsageFrom,
		IssueAfterReset:         model.IssueAfterReset,
		IssueAfterResetPriority: model.IssueAfterResetPriority,
		IsSoftLimit:             model.IsSoftLimit,
		UsagePeriod:             model.UsagePeriod,
		CurrentUsagePeriod:      &currentPeriod,
		PreserveOverageAtReset:  model.PreserveOverageAtReset,
		ActiveFrom:              model.ActiveFrom,
		ActiveTo:                model.ActiveTo,
	}, nil
}

func (c *connector) AfterCreate(ctx context.Context, end *entitlement.Entitlement) error {
	metered, err := ParseFromGenericEntitlement(end)
	if err != nil {
		return err
	}

	// issue default grants
	if metered.HasDefaultGrant() {
		if metered.IssueAfterReset == nil {
			return fmt.Errorf("inconsistency error: entitlement %s should have default grant but has no IssueAfterReset", metered.ID)
		}

		effectiveAt := metered.CurrentUsagePeriod.From
		amountToIssue := metered.IssueAfterReset.Amount
		_, err := c.grantConnector.CreateGrant(ctx, models.NamespacedID{
			Namespace: metered.Namespace,
			ID:        metered.ID,
		}, credit.CreateGrantInput{
			Amount:      amountToIssue,
			Priority:    defaultx.WithDefault(metered.IssueAfterReset.Priority, DefaultIssueAfterResetPriority),
			EffectiveAt: effectiveAt,
			Expiration:  nil, // We don't want to expire the grant
			// These two in conjunction make the grant always have `amountToIssue` balance after a reset
			ResetMaxRollover: amountToIssue,
			ResetMinRollover: amountToIssue,
			Annotations: models.Annotations{
				IssueAfterResetMetaTag: true,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
