package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementConnector interface {
	// Entitlement Management
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error)
}

type entitlementConnector struct {
	ebc EntitlementBalanceConnector
	edb EntitlementDBConnector
	fc  productcatalog.FeatureConnector
}

func NewEntitlementConnector(
	ebc EntitlementBalanceConnector,
	edb EntitlementDBConnector,
	fc productcatalog.FeatureConnector,
) EntitlementConnector {
	return &entitlementConnector{
		ebc: ebc,
		edb: edb,
		fc:  fc,
	}
}

func (c *entitlementConnector) CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error) {
	// TODO: check if the feature exists, if it is compatible with the type, etc....
	_, err := c.fc.GetFeature(ctx, models.NamespacedID{Namespace: input.Namespace, ID: input.FeatureID})
	if err != nil {
		return Entitlement{}, &productcatalog.FeatureNotFoundError{ID: input.FeatureID}
	}
	currentEntitlements, err := c.edb.GetEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey))
	if err != nil {
		return Entitlement{}, fmt.Errorf("failed to get entitlements of subject: %w", err)
	}
	for _, ent := range currentEntitlements {
		if ent.FeatureID == input.FeatureID {
			return Entitlement{}, &EntitlementAlreadyExistsError{EntitlementID: ent.ID, FeatureID: input.FeatureID, SubjectKey: input.SubjectKey}
		}
	}

	// FIXME: Add default value elsewhere
	input.MeasureUsageFrom = time.Now().Truncate(time.Minute)
	ent, err := c.edb.CreateEntitlement(ctx, CreateEntitlementDBInputs(input))
	return *ent, err
}

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error) {
	return c.edb.GetEntitlementsOfSubject(ctx, namespace, subjectKey)
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error) {
	// TODO: different entitlement types
	balance, err := c.ebc.GetEntitlementBalance(ctx, entitlementId, at)

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
