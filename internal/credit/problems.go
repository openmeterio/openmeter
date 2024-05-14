package credit

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

type LedgerAlreadyExistsProblemExtensionMetadata struct {
	ConflictingEntity Ledger `json:"conflictingEntity"`
}

type LedgerAlreadyExistsProblemResponse models.StatusProblemWithExtension[LedgerAlreadyExistsProblemExtensionMetadata]

func NewLedgerAlreadyExistsProblem(ctx context.Context, err error, existingEntry Ledger) models.Problem {
	return models.NewStatusProblemWithExtension(
		ctx, err, http.StatusConflict,
		LedgerAlreadyExistsProblemExtensionMetadata{
			ConflictingEntity: existingEntry,
		},
	)
}
