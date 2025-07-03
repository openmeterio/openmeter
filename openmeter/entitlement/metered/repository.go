package meteredentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type UsageResetRepo interface {
	Save(ctx context.Context, usageResetTime UsageResetUpdate) error
}

type UsageResetNotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e UsageResetNotFoundError) Error() string {
	return fmt.Sprintf("usage reset not found for entitlement %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}

type UsageResetUpdate struct {
	models.NamespacedModel
	ResetTime           time.Time
	Anchor              time.Time
	EntitlementID       string
	UsagePeriodInterval isodate.String
}

func (u UsageResetUpdate) Validate() error {
	if u.UsagePeriodInterval == "" {
		return fmt.Errorf("usage period interval is required")
	}

	if _, err := u.UsagePeriodInterval.Parse(); err != nil {
		return fmt.Errorf("invalid usage period interval: %w", err)
	}

	return nil
}
