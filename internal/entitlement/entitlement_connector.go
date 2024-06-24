package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEntitlementsOrderBy string

const (
	ListEntitlementsOrderByCreatedAt ListEntitlementsOrderBy = "created_at"
	ListEntitlementsOrderByUpdatedAt ListEntitlementsOrderBy = "updated_at"
)

type ListEntitlementsParams struct {
	Namespace string
	Limit     int
	Offset    int
	OrderBy   ListEntitlementsOrderBy
}

type EntitlementConnector interface {
	// Entitlement Management
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error)

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error)
}

type entitlementConnector struct {
	entitlementBalanceConnector EntitlementBalanceConnector
	entitlementRepo             EntitlementRepo
	featureConnector            productcatalog.FeatureConnector
}

func NewEntitlementConnector(
	entitlementBalanceConnector EntitlementBalanceConnector,
	entitlementRepo EntitlementRepo,
	featureConnector productcatalog.FeatureConnector,
) EntitlementConnector {
	return &entitlementConnector{
		entitlementBalanceConnector: entitlementBalanceConnector,
		entitlementRepo:             entitlementRepo,
		featureConnector:            featureConnector,
	}
}

func (c *entitlementConnector) CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error) {
	// TODO: check if the feature exists, if it is compatible with the type, etc....
	feature, err := c.featureConnector.GetFeature(ctx, models.NamespacedID{Namespace: input.Namespace, ID: input.FeatureID})
	if err != nil {
		return Entitlement{}, &productcatalog.FeatureNotFoundError{ID: input.FeatureID}
	}
	if feature.ArchivedAt != nil {
		return Entitlement{}, &models.GenericUserError{Message: "Feature is archived"}
	}
	currentEntitlements, err := c.entitlementRepo.GetEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey))
	if err != nil {
		return Entitlement{}, fmt.Errorf("failed to get entitlements of subject: %w", err)
	}
	for _, ent := range currentEntitlements {
		if ent.FeatureID == input.FeatureID {
			return Entitlement{}, &EntitlementAlreadyExistsError{EntitlementID: ent.ID, FeatureID: input.FeatureID, SubjectKey: input.SubjectKey}
		}
	}

	nextReset, err := input.UsagePeriod.NextAfter(time.Now())
	if err != nil {
		return Entitlement{}, fmt.Errorf("failed to calculate next reset: %w", err)
	}

	ent, err := c.entitlementRepo.CreateEntitlement(ctx, EntitlementRepoCreateEntitlementInputs{
		Namespace:  input.Namespace,
		FeatureID:  input.FeatureID,
		SubjectKey: input.SubjectKey,
		// FIXME: Add default value elsewhere
		MeasureUsageFrom: time.Now().Truncate(time.Minute),
		UsagePeriod: RecurrenceWithNextReset{
			Recurrence: input.UsagePeriod,
			NextReset:  nextReset,
		},
	})
	return *ent, err
}

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error) {
	return c.entitlementRepo.GetEntitlementsOfSubject(ctx, namespace, subjectKey)
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error) {
	// TODO: different entitlement types
	balance, err := c.entitlementBalanceConnector.GetEntitlementBalance(ctx, entitlementId, at)

	if err != nil {
		return EntitlementValue{}, err
	}

	return EntitlementValue{
		HasAccess: balance.Balance > 0,
		Balance:   balance.Balance,
		Usage:     balance.UsageInPeriod,
		Overage:   balance.Overage,
	}, nil
}

func (c *entitlementConnector) ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error) {
	return c.entitlementRepo.ListEntitlements(ctx, params)
}
