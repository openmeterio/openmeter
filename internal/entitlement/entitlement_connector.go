package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api/types"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementConnector interface {
	// Entitlement Management
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error)
}

type entitlementConnector struct {
	entitlementBalanceConnector EntitlementBalanceConnector
	entitlementRepo             EntitlementRepo
	featureConnector            productcatalog.FeatureConnector
	billingConnector            subject.BillingConnector
}

func NewEntitlementConnector(
	entitlementBalanceConnector EntitlementBalanceConnector,
	entitlementRepo EntitlementRepo,
	featureConnector productcatalog.FeatureConnector,
	billingConnector subject.BillingConnector,
) EntitlementConnector {
	return &entitlementConnector{
		entitlementBalanceConnector: entitlementBalanceConnector,
		entitlementRepo:             entitlementRepo,
		featureConnector:            featureConnector,
		billingConnector:            billingConnector,
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

	var nextRecurrence time.Time

	if input.UsagePeriod.Interval == types.RecurringPeriodBilling {
		billingPeriod, err := c.billingConnector.GetBillingPeriodOfSubject(input.Namespace, input.SubjectKey)
		if err != nil {
			// TODO: have a specific 400 error for this
			return Entitlement{}, err
		}

		if !billingPeriod.PeriodEnd.Before(time.Now()) {
			return Entitlement{}, fmt.Errorf("billing period is already over for subject: %s", billingPeriod.PeriodEnd.Format(time.RFC3339))
		}

		nextRecurrence = billingPeriod.PeriodEnd
	} else {
		// TODO: billing period handling
		nextRecurrence, err = input.UsagePeriod.NextRecurrence()
		if err != nil {
			return Entitlement{}, err
		}

		if !nextRecurrence.After(time.Now()) {
			return Entitlement{}, &models.GenericUserError{Message: "Next recurrence must be in the future"}
		}
	}

	// FIXME: Add default value elsewhere
	input.MeasureUsageFrom = time.Now().Truncate(time.Minute)
	ent, err := c.entitlementRepo.CreateEntitlement(ctx, EntitlementRepoCreateEntitlementInputs{
		Namespace:        input.Namespace,
		FeatureID:        input.FeatureID,
		MeasureUsageFrom: input.MeasureUsageFrom,
		SubjectKey:       input.SubjectKey,
		UsagePeriod: types.RecurringPeriod{
			RecurringPeriodCreateInputs: input.UsagePeriod,
			NextRecurrence:              nextRecurrence,
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
