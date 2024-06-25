package entitlement

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementRepo interface {
	// Entitlement Management
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	CreateEntitlement(ctx context.Context, entitlement CreateEntitlementInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error)

	//FIXME: This is a terrbile hack
	LockEntitlementForTx(ctx context.Context, entitlementID models.NamespacedID) error

	entutils.TxCreator
	entutils.TxUser[EntitlementRepo]
}
