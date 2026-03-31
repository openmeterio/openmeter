package invoicesyncadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripeinvoicesyncop"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripeinvoicesyncplan"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client *entdb.Client
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}
	return nil
}

func New(config Config) (*Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &Adapter{db: config.Client}, nil
}

var _ invoicesync.Adapter = (*Adapter)(nil)

type Adapter struct {
	db *entdb.Client
}

func (a *Adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *Adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *Adapter {
	txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &Adapter{db: txClient.Client()}
}

func (a *Adapter) Self() *Adapter {
	return a
}

func (a *Adapter) CreateSyncPlan(ctx context.Context, plan invoicesync.SyncPlan) (invoicesync.SyncPlan, error) {
	// Use TransactingRepo to join any existing transaction (e.g., the billing state machine's
	// transaction) so the sync plan FK to billing_invoices sees uncommitted rows.
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *Adapter) (invoicesync.SyncPlan, error) {
		created, err := tx.db.AppStripeInvoiceSyncPlan.Create().
			SetNamespace(plan.Namespace).
			SetInvoiceID(plan.InvoiceID).
			SetAppID(plan.AppID).
			SetSessionID(plan.SessionID).
			SetPhase(plan.Phase).
			SetStatus(invoicesync.PlanStatusPending).
			Save(ctx)
		if err != nil {
			return invoicesync.SyncPlan{}, fmt.Errorf("creating sync plan: %w", err)
		}

		plan.ID = created.ID
		plan.CreatedAt = created.CreatedAt
		plan.UpdatedAt = created.UpdatedAt
		plan.Status = invoicesync.PlanStatusPending

		// Bulk-create all operations
		builders := make([]*entdb.AppStripeInvoiceSyncOpCreate, len(plan.Operations))
		for i, op := range plan.Operations {
			builders[i] = tx.db.AppStripeInvoiceSyncOp.Create().
				SetPlanID(created.ID).
				SetSequence(op.Sequence).
				SetType(op.Type).
				SetPayload(op.Payload).
				SetIdempotencyKey(op.IdempotencyKey).
				SetStatus(invoicesync.OpStatusPending)
		}

		createdOps, err := tx.db.AppStripeInvoiceSyncOp.CreateBulk(builders...).Save(ctx)
		if err != nil {
			return invoicesync.SyncPlan{}, fmt.Errorf("creating sync operations: %w", err)
		}

		for i, createdOp := range createdOps {
			plan.Operations[i].ID = createdOp.ID
			plan.Operations[i].PlanID = created.ID
			plan.Operations[i].CreatedAt = createdOp.CreatedAt
			plan.Operations[i].UpdatedAt = createdOp.UpdatedAt
			plan.Operations[i].Status = invoicesync.OpStatusPending
		}

		return plan, nil
	})
}

func (a *Adapter) GetSyncPlan(ctx context.Context, planID string) (*invoicesync.SyncPlan, error) {
	row, err := a.db.AppStripeInvoiceSyncPlan.Query().
		Where(appstripeinvoicesyncplan.ID(planID)).
		WithOperations(func(q *entdb.AppStripeInvoiceSyncOpQuery) {
			q.Order(entdb.Asc(appstripeinvoicesyncop.FieldSequence))
		}).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting sync plan: %w", err)
	}

	return mapPlanFromDB(row), nil
}

func (a *Adapter) GetActiveSyncPlanByInvoice(ctx context.Context, namespace, invoiceID string, phase invoicesync.SyncPlanPhase) (*invoicesync.SyncPlan, error) {
	// Use Order + First instead of Only: if a crash leaves two active plans for the same
	// invoice+phase (no unique constraint on status), we pick the most recent one.
	row, err := a.db.AppStripeInvoiceSyncPlan.Query().
		Where(
			appstripeinvoicesyncplan.Namespace(namespace),
			appstripeinvoicesyncplan.InvoiceID(invoiceID),
			appstripeinvoicesyncplan.Phase(phase),
			appstripeinvoicesyncplan.StatusIn(invoicesync.PlanStatusPending, invoicesync.PlanStatusExecuting),
		).
		WithOperations(func(q *entdb.AppStripeInvoiceSyncOpQuery) {
			q.Order(entdb.Asc(appstripeinvoicesyncop.FieldSequence))
		}).
		Order(entdb.Desc(appstripeinvoicesyncplan.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting active sync plan: %w", err)
	}

	return mapPlanFromDB(row), nil
}

func (a *Adapter) GetActiveSyncPlansByInvoice(ctx context.Context, namespace, invoiceID string) ([]invoicesync.SyncPlan, error) {
	rows, err := a.db.AppStripeInvoiceSyncPlan.Query().
		Where(
			appstripeinvoicesyncplan.Namespace(namespace),
			appstripeinvoicesyncplan.InvoiceID(invoiceID),
			appstripeinvoicesyncplan.StatusIn(invoicesync.PlanStatusPending, invoicesync.PlanStatusExecuting),
		).
		Order(entdb.Desc(appstripeinvoicesyncplan.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting active sync plans: %w", err)
	}

	plans := make([]invoicesync.SyncPlan, len(rows))
	for i, row := range rows {
		plans[i] = *mapPlanFromDB(row)
	}
	return plans, nil
}

func (a *Adapter) GetNextPendingOperation(ctx context.Context, planID string) (*invoicesync.SyncOperation, error) {
	row, err := a.db.AppStripeInvoiceSyncOp.Query().
		Where(
			appstripeinvoicesyncop.PlanID(planID),
			appstripeinvoicesyncop.Status(invoicesync.OpStatusPending),
		).
		Order(entdb.Asc(appstripeinvoicesyncop.FieldSequence)).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting next pending operation: %w", err)
	}

	op := mapOpFromDB(row)
	return &op, nil
}

func (a *Adapter) CompleteOperation(ctx context.Context, opID string, stripeResponse json.RawMessage) error {
	now := clock.Now().UTC()
	err := a.db.AppStripeInvoiceSyncOp.UpdateOneID(opID).
		SetStatus(invoicesync.OpStatusCompleted).
		SetStripeResponse(stripeResponse).
		SetCompletedAt(now).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("completing operation: %w", err)
	}
	return nil
}

func (a *Adapter) FailOperation(ctx context.Context, opID string, errMsg string) error {
	now := clock.Now().UTC()
	err := a.db.AppStripeInvoiceSyncOp.UpdateOneID(opID).
		SetStatus(invoicesync.OpStatusFailed).
		SetError(errMsg).
		SetCompletedAt(now).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failing operation: %w", err)
	}
	return nil
}

func (a *Adapter) UpdatePlanStatus(ctx context.Context, planID string, status invoicesync.PlanStatus, errMsg *string) error {
	update := a.db.AppStripeInvoiceSyncPlan.UpdateOneID(planID).
		SetStatus(status)

	if errMsg != nil {
		update = update.SetError(*errMsg)
	}

	if status == invoicesync.PlanStatusCompleted || status == invoicesync.PlanStatusFailed {
		now := clock.Now().UTC()
		update = update.SetCompletedAt(now)
	}

	if err := update.Exec(ctx); err != nil {
		return fmt.Errorf("updating plan status: %w", err)
	}
	return nil
}

func (a *Adapter) CompletePlan(ctx context.Context, planID string) error {
	return a.UpdatePlanStatus(ctx, planID, invoicesync.PlanStatusCompleted, nil)
}

func (a *Adapter) FailPlan(ctx context.Context, planID string, errMsg string) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *Adapter) error {
		now := clock.Now().UTC()

		// Cancel remaining pending operations so they won't be picked up on resume
		_, err := tx.db.AppStripeInvoiceSyncOp.Update().
			Where(
				appstripeinvoicesyncop.PlanID(planID),
				appstripeinvoicesyncop.Status(invoicesync.OpStatusPending),
			).
			SetStatus(invoicesync.OpStatusFailed).
			SetError("plan failed: " + errMsg).
			SetCompletedAt(now).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("canceling pending operations: %w", err)
		}

		// Update plan status
		return tx.db.AppStripeInvoiceSyncPlan.UpdateOneID(planID).
			SetStatus(invoicesync.PlanStatusFailed).
			SetError(errMsg).
			SetCompletedAt(now).
			Exec(ctx)
	})
}

func mapPlanFromDB(row *entdb.AppStripeInvoiceSyncPlan) *invoicesync.SyncPlan {
	plan := &invoicesync.SyncPlan{
		ID:          row.ID,
		Namespace:   row.Namespace,
		InvoiceID:   row.InvoiceID,
		AppID:       row.AppID,
		SessionID:   row.SessionID,
		Phase:       row.Phase,
		Status:      row.Status,
		Error:       row.Error,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		CompletedAt: row.CompletedAt,
	}

	if row.Edges.Operations != nil {
		plan.Operations = make([]invoicesync.SyncOperation, len(row.Edges.Operations))
		for i, op := range row.Edges.Operations {
			plan.Operations[i] = mapOpFromDB(op)
		}
	}

	return plan
}

func mapOpFromDB(row *entdb.AppStripeInvoiceSyncOp) invoicesync.SyncOperation {
	return invoicesync.SyncOperation{
		ID:             row.ID,
		PlanID:         row.PlanID,
		Sequence:       row.Sequence,
		Type:           row.Type,
		Payload:        row.Payload,
		IdempotencyKey: row.IdempotencyKey,
		Status:         row.Status,
		StripeResponse: row.StripeResponse,
		Error:          row.Error,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		CompletedAt:    row.CompletedAt,
	}
}
