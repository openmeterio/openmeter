package credit

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

type LedgerAlreadyExistsProblemResponse struct {
	*models.StatusProblem
	ConflictingEntity Ledger `json:"conflictingEntity"`
}

func (p *LedgerAlreadyExistsProblemResponse) Respond(w http.ResponseWriter, r *http.Request) {
	models.RespondProblem(p, w, r)
}

func NewLedgerAlreadyExistsProblem(ctx context.Context, err error, existingEntry Ledger) models.Problem {
	return &LedgerAlreadyExistsProblemResponse{
		StatusProblem:     models.NewStatusProblem(ctx, err, http.StatusConflict),
		ConflictingEntity: existingEntry,
	}
}
