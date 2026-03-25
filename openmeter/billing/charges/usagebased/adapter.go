package usagebased

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	RealizationRunAdapter
	RealizationRunCreditAllocationAdapter
	ChargeAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	CreateCharges(ctx context.Context, charges CreateInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, input GetByIDInput) (Charge, error)
	UpdateStatus(ctx context.Context, input UpdateStatusInput) (ChargeBase, error)
}

type RealizationRunAdapter interface {
	CreateRealizationRun(ctx context.Context, chargeID meta.ChargeID, input CreateRealizationRunInput) (RealizationRunBase, error)
	UpdateRealizationRun(ctx context.Context, input UpdateRealizationRunInput) (RealizationRunBase, error)
}

type RealizationRunCreditAllocationAdapter interface {
	CreateRunCreditAllocations(ctx context.Context, runID RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error)
}
