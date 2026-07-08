package usagebased

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Status string

const (
	StatusCreated Status = Status(meta.ChargeStatusCreated)

	// Active status and substates
	StatusActive Status = Status(meta.ChargeStatusActive)

	// Deprecated: use StatusActiveRealizationStarted. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActivePartialInvoiceStarted Status = "active.partial_invoice.started"
	// Deprecated: use StatusActiveRealizationWaitingForCollection. Kept
	// temporarily so persisted rows can load until the SQL migration rewrites
	// legacy statuses.
	StatusActivePartialInvoiceWaitingForCollection Status = "active.partial_invoice.waiting_for_collection"
	// Deprecated: use StatusActiveRealizationProcessing. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActivePartialInvoiceProcessing Status = "active.partial_invoice.processing"
	// Deprecated: use StatusActiveRealizationIssuing. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActivePartialInvoiceIssuing Status = "active.partial_invoice.issuing"
	// Deprecated: use StatusActiveRealizationCompleted. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActivePartialInvoiceCompleted         Status = "active.partial_invoice.completed"
	StatusActiveRealizationStarted              Status = "active.realization.started"
	StatusActiveRealizationWaitingForCollection Status = "active.realization.waiting_for_collection"
	StatusActiveRealizationProcessing           Status = "active.realization.processing"
	StatusActiveRealizationIssuing              Status = "active.realization.issuing"
	StatusActiveRealizationCompleted            Status = "active.realization.completed"
	// Deprecated: use StatusActiveRealizationStarted. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActiveFinalRealizationStarted Status = "active.final_realization.started"
	// Deprecated: use StatusActiveRealizationWaitingForCollection. Kept
	// temporarily so persisted rows can load until the SQL migration rewrites
	// legacy statuses.
	StatusActiveFinalRealizationWaitingForCollection Status = "active.final_realization.waiting_for_collection"
	// Deprecated: use StatusActiveRealizationProcessing. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActiveFinalRealizationProcessing Status = "active.final_realization.processing"
	// Deprecated: use StatusActiveRealizationIssuing. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActiveFinalRealizationIssuing Status = "active.final_realization.issuing"
	// Deprecated: use StatusActiveRealizationCompleted. Kept temporarily so
	// persisted rows can load until the SQL migration rewrites legacy statuses.
	StatusActiveFinalRealizationCompleted Status = "active.final_realization.completed"
	StatusActiveAwaitingPaymentSettlement Status = "active.awaiting_payment_settlement"

	StatusFinal   Status = Status(meta.ChargeStatusFinal)
	StatusDeleted Status = Status(meta.ChargeStatusDeleted)
)

var legacyStatusMap = map[Status]Status{
	StatusActivePartialInvoiceStarted:                StatusActiveRealizationStarted,
	StatusActivePartialInvoiceWaitingForCollection:   StatusActiveRealizationWaitingForCollection,
	StatusActivePartialInvoiceProcessing:             StatusActiveRealizationProcessing,
	StatusActivePartialInvoiceIssuing:                StatusActiveRealizationIssuing,
	StatusActivePartialInvoiceCompleted:              StatusActiveRealizationCompleted,
	StatusActiveFinalRealizationStarted:              StatusActiveRealizationStarted,
	StatusActiveFinalRealizationWaitingForCollection: StatusActiveRealizationWaitingForCollection,
	StatusActiveFinalRealizationProcessing:           StatusActiveRealizationProcessing,
	StatusActiveFinalRealizationIssuing:              StatusActiveRealizationIssuing,
	StatusActiveFinalRealizationCompleted:            StatusActiveRealizationCompleted,
}

// NormalizeLegacyStatus maps persisted pre-unification realization statuses to
// the canonical active.realization.* status branch. We do this at state-machine
// load time so old rows keep working without keeping legacy states in the
// lifecycle graph. The legacy values stay accepted until a follow-up SQL
// migration rewrites status_detailed in storage and the enum can be tightened.
func NormalizeLegacyStatus(status Status) Status {
	if normalized, ok := legacyStatusMap[status]; ok {
		return normalized
	}

	return status
}

// mutableRealizationStatuses are states where the current realization can still
// be rebuilt by period changes instead of touching immutable invoice or ledger
// records.
var mutableRealizationStatuses = []Status{
	StatusActiveRealizationStarted,
	StatusActiveRealizationWaitingForCollection,
	StatusActiveRealizationProcessing,
}

func IsMutableRealizationStatus(status Status) bool {
	return slices.Contains(mutableRealizationStatuses, NormalizeLegacyStatus(status))
}

func (Status) Values() []string {
	return []string{
		string(StatusCreated),
		string(StatusActive),
		string(StatusActivePartialInvoiceStarted),
		string(StatusActivePartialInvoiceWaitingForCollection),
		string(StatusActivePartialInvoiceProcessing),
		string(StatusActivePartialInvoiceIssuing),
		string(StatusActivePartialInvoiceCompleted),
		string(StatusActiveRealizationStarted),
		string(StatusActiveRealizationWaitingForCollection),
		string(StatusActiveRealizationProcessing),
		string(StatusActiveRealizationIssuing),
		string(StatusActiveRealizationCompleted),
		string(StatusActiveFinalRealizationStarted),
		string(StatusActiveFinalRealizationWaitingForCollection),
		string(StatusActiveFinalRealizationProcessing),
		string(StatusActiveFinalRealizationIssuing),
		string(StatusActiveFinalRealizationCompleted),
		string(StatusActiveAwaitingPaymentSettlement),
		string(StatusFinal),
		string(StatusDeleted),
	}
}

func (s Status) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid status: %s", s))
	}
	return nil
}

func (s Status) ToMetaChargeStatus() (meta.ChargeStatus, error) {
	if err := s.Validate(); err != nil {
		return meta.ChargeStatusCreated, err
	}

	return meta.DetailedStatusToMetaStatus(string(s))
}
