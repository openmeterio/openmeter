package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Get credit balance, GET /api/v1/ledgers/{ledgerID}/history
func (a *Router) GetCreditHistory(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, params api.GetCreditHistoryParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getCreditLedger")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get Ledger
	ledgerEntries, err := a.config.CreditConnector.GetHistory(
		ctx, namespace, ledgerID,
		params.From,
		defaultx.WithDefault(params.To, time.Now()),
		defaultx.WithDefault(params.Limit, api.DefaultCreditsQueryLimit),
	)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, ledgerEntries)
}

// CreateLedger POS /api/v1/ledgers
func (a *Router) CreateLedger(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "createCreditLedger")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Parse request body
	createLedgerArgs := &api.CreateLedger{}
	if err := render.DecodeJSON(r.Body, createLedgerArgs); err != nil {
		err := fmt.Errorf("decode json: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)

		return
	}

	if createLedgerArgs.Subject == "" {
		err := fmt.Errorf("subject must be non-empty when creating a new ledger")
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	ledger, err := a.config.CreditConnector.CreateLedger(ctx, namespace, credit.Ledger{
		Subject:  createLedgerArgs.Subject,
		Metadata: createLedgerArgs.Metadata,
	}, false)
	if err != nil {

		if existsError, ok := err.(*credit.LedgerAlreadyExistsError); ok {
			err := fmt.Errorf("ledger already exists for subject: %s, existing ledger %s.%s",
				existsError.Subject,
				existsError.Namespace,
				existsError.LedgerID.String())
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, ledger)

}

// ListLedgers GET /api/v1/ledgers?subject=X&offset=Y&limit=Z
func (a *Router) ListLedgers(w http.ResponseWriter, r *http.Request, params api.ListLedgersParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "createCreditLedger")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	ledgers, err := a.config.CreditConnector.ListLedgers(ctx, namespace,
		credit.ListLedgersParams{
			Subjects: defaultx.WithDefault(params.Subject, nil),
			Offset:   defaultx.WithDefault(params.Offset, 0),
			Limit:    defaultx.WithDefault(params.Limit, 0),
		})
	if err != nil && !db.IsNotFound(err) {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Response
	list := slicesx.Map(ledgers, func(ledger credit.Ledger) render.Renderer {
		return ledger
	})
	_ = render.RenderList(w, r, list)
}
