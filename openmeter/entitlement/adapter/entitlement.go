package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	db_entitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	db_feature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	db_usagereset "github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type entitlementDBAdapter struct {
	db *db.Client
}

type repo interface {
	entitlement.EntitlementRepo
	balanceworker.BalanceWorkerRepository
}

var _ repo = (*entitlementDBAdapter)(nil)

var _ interface {
	transaction.Creator
	entutils.TxUser[*entitlementDBAdapter]
} = (*entitlementDBAdapter)(nil)

func NewPostgresEntitlementRepo(db *db.Client) *entitlementDBAdapter {
	return &entitlementDBAdapter{
		db: db,
	}
}

func (a *entitlementDBAdapter) GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			res, err := withAllUsageResets(repo.db.Entitlement.Query(), []string{entitlementID.Namespace}).
				WithCustomer(func(q *db.CustomerQuery) {
					customeradapter.WithSubjects(q, clock.Now())
				}).
				Where(
					db_entitlement.ID(entitlementID.ID),
					db_entitlement.Namespace(entitlementID.Namespace),
					db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()),
				).
				First(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
				}
				return nil, err
			}

			return repo.mapEntitlementEntity(res)
		},
	)
}

func (a *entitlementDBAdapter) GetActiveEntitlementOfCustomerAt(ctx context.Context, namespace string, customerID string, featureKey string, at time.Time) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			res, err := withAllUsageResets(repo.db.Entitlement.Query(), []string{namespace}).
				WithCustomer(func(q *db.CustomerQuery) {
					customeradapter.WithSubjects(q, at)
				}).
				Where(EntitlementActiveAt(at)...).
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtGT(at), db_entitlement.DeletedAtIsNil()),
					db_entitlement.HasCustomerWith(
						customerdb.Namespace(namespace),
						customerNotDeletedAt(at),
						customerdb.ID(customerID),
					),
					db_entitlement.Namespace(namespace),
					db_entitlement.FeatureKey(featureKey),
				).
				First(ctx) // FIXME: to better enforce consistency we should not use .First() but assert that there is only one result!
			if err != nil {
				if db.IsNotFound(err) {
					return nil, &entitlement.NotFoundError{
						EntitlementID: models.NamespacedID{
							Namespace: namespace,
							ID:        featureKey,
						},
					}
				}
				return nil, err
			}

			return repo.mapEntitlementEntity(res)
		},
	)
}

func (a *entitlementDBAdapter) CreateEntitlement(ctx context.Context, ent entitlement.CreateEntitlementRepoInputs) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo[*entitlement.Entitlement, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			now := clock.Now().UTC()

			query := repo.db.Customer.Query().Where(
				customerdb.Namespace(ent.Namespace),
				customerdb.ID(ent.UsageAttribution.ID),
				customerdb.DeletedAtIsNil(),
			)

			query = customeradapter.WithSubjects(query, now)

			cus, err := query.Only(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, models.NewGenericNotFoundError(
						fmt.Errorf("customer with id %s not found in %s namespace", ent.UsageAttribution.ID, ent.Namespace),
					)
				}

				return nil, fmt.Errorf("failed to resolve customer: %w", err)
			}

			cmd := repo.db.Entitlement.Create().
				SetEntitlementType(db_entitlement.EntitlementType(ent.EntitlementType)).
				SetNamespace(ent.Namespace).
				SetFeatureID(ent.FeatureID).
				SetMetadata(ent.Metadata).
				SetCustomerID(cus.ID).
				SetFeatureKey(ent.FeatureKey).
				SetNillableMeasureUsageFrom(ent.MeasureUsageFrom).
				SetNillableIssueAfterReset(ent.IssueAfterReset).
				SetNillableIssueAfterResetPriority(ent.IssueAfterResetPriority).
				SetNillablePreserveOverageAtReset(ent.PreserveOverageAtReset).
				SetNillableIsSoftLimit(ent.IsSoftLimit).
				SetNillableActiveFrom(ent.ActiveFrom).
				SetNillableActiveTo(ent.ActiveTo)

			if ent.Annotations != nil {
				cmd.SetAnnotations(ent.Annotations)
			}

			if ent.UsagePeriod != nil {
				usagePeriod := ent.UsagePeriod.GetValue()

				cmd.SetNillableUsagePeriodAnchor(&usagePeriod.Anchor).
					SetNillableUsagePeriodInterval(usagePeriod.Interval.ISOStringPtrOrNil())
			}

			if ent.CurrentUsagePeriod != nil {
				cmd.SetNillableCurrentUsagePeriodStart(&ent.CurrentUsagePeriod.From).
					SetNillableCurrentUsagePeriodEnd(&ent.CurrentUsagePeriod.To)
			}

			if ent.Config != nil {
				cmd.SetConfig(ent.Config)
			}

			res, err := cmd.Save(ctx)
			if err != nil {
				return nil, err
			}

			// Query the created entitlement back with customer and subject edges loaded
			entWithEdges, err := repo.db.Entitlement.Query().
				WithCustomer(func(q *db.CustomerQuery) {
					customeradapter.WithSubjects(q, now)
				}).
				Where(db_entitlement.ID(res.ID)).
				Only(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, models.NewGenericNotFoundError(
						fmt.Errorf("entitlement with id %s not found in %s namespace", res.ID, res.Namespace),
					)
				}
				return nil, fmt.Errorf("failed to query created entitlement with edges: %w", err)
			}

			return repo.mapEntitlementEntity(entWithEdges)
		},
	)
}

func (a *entitlementDBAdapter) DeleteEntitlement(ctx context.Context, entitlementID models.NamespacedID, at time.Time) error {
	_, err := entutils.TransactingRepo[*entitlement.Entitlement, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			affectedCount, err := repo.db.Entitlement.Update().
				Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
				SetDeletedAt(at).
				Save(ctx)
			if err != nil {
				return nil, err
			}
			if affectedCount == 0 {
				return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
			}
			return nil, nil
		},
	)
	return err
}

func (a *entitlementDBAdapter) DeactivateEntitlement(ctx context.Context, entitlementID models.NamespacedID, at time.Time) error {
	_, err := entutils.TransactingRepo[interface{}, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (interface{}, error) {
			ent, err := repo.GetEntitlement(ctx, entitlementID)
			if err != nil {
				return nil, err
			}

			if ent.ActiveTo != nil {
				return nil, fmt.Errorf("entitlement %s is already deactivated", entitlementID.ID)
			}

			return nil, repo.db.Entitlement.UpdateOneID(ent.ID).SetActiveTo(at).Exec(ctx)
		},
	)

	return err
}

// TODO[OM-1009]: This returns all the entitlements even the expired ones, for billing we would need to have a range for
// the batch ingested events. Let's narrow down the list of entitlements active during that period.
func (a *entitlementDBAdapter) ListEntitlementsAffectedByIngestEvents(ctx context.Context, eventFilters []balanceworker.IngestEventQueryFilter) ([]balanceworker.ListAffectedEntitlementsResponse, error) {
	return entutils.TransactingRepo[[]balanceworker.ListAffectedEntitlementsResponse, *entitlementDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]balanceworker.ListAffectedEntitlementsResponse, error) {
			if len(eventFilters) == 0 {
				return nil, fmt.Errorf("no eventFilters provided")
			}

			result := make([]balanceworker.ListAffectedEntitlementsResponse, 0)

			for _, pair := range eventFilters {
				entities, err := repo.db.Entitlement.Query().
					Where(
						db_entitlement.Namespace(pair.Namespace),
						db_entitlement.HasCustomerWith(
							customerdb.Namespace(pair.Namespace),
							customerNotDeletedAt(clock.Now()),
							customerdb.HasSubjectsWith(
								customersubjectsdb.SubjectKey(pair.EventSubject),
								customersubjectsdb.DeletedAtIsNil(),
							),
						),
						db_entitlement.HasFeatureWith(db_feature.MeterSlugIn(pair.MeterSlugs...)),
					).
					WithFeature().
					WithCustomer(
						func(q *db.CustomerQuery) {
							customeradapter.WithSubjects(q, clock.Now())
						},
					).
					All(ctx)
				if err != nil {
					return nil, err
				}

				for _, e := range entities {
					if e.Edges.Customer == nil {
						return nil, fmt.Errorf("entitlement %s has no customer", e.ID)
					}

					result = append(result, balanceworker.ListAffectedEntitlementsResponse{
						Namespace:     e.Namespace,
						EntitlementID: e.ID,
						SubjectKey:    pair.EventSubject,
						CustomerID:    e.Edges.Customer.ID,
						CreatedAt:     e.CreatedAt.UTC(),
						DeletedAt:     convert.SafeToUTC(e.DeletedAt),
						ActiveFrom:    convert.SafeToUTC(e.ActiveFrom),
						ActiveTo:      convert.SafeToUTC(e.ActiveTo),
						MeterSlug:     e.Edges.Feature.MeterSlug,
					})
				}
			}

			return result, nil
		})
}

func (a *entitlementDBAdapter) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.Result[entitlement.Entitlement], error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (pagination.Result[entitlement.Entitlement], error) {
			now := clock.Now().UTC()

			query := repo.db.Entitlement.Query().WithCustomer(func(q *db.CustomerQuery) {
				customeradapter.WithSubjects(q, now)
			})

			if len(params.Namespaces) > 0 {
				query = query.Where(db_entitlement.NamespaceIn(params.Namespaces...))
			}

			query = withAllUsageResets(query, params.Namespaces)

			if len(params.SubjectKeys) > 0 {
				query = query.Where(
					db_entitlement.HasCustomerWith(
						customerdb.HasSubjectsWith(
							customersubjectsdb.SubjectKeyIn(params.SubjectKeys...),
							customersubjectsdb.DeletedAtIsNil(),
						),
						customerNotDeletedAt(now),
					),
				)
			}

			if len(params.CustomerKeys) > 0 {
				query = query.Where(db_entitlement.HasCustomerWith(
					customerdb.KeyIn(params.CustomerKeys...),
				))
			}

			if len(params.CustomerIDs) > 0 {
				query = query.Where(db_entitlement.HasCustomerWith(
					customerdb.IDIn(params.CustomerIDs...),
				))
			}

			if len(params.EntitlementTypes) > 0 {
				query = query.Where(db_entitlement.EntitlementTypeIn(slicesx.Map(params.EntitlementTypes, func(t entitlement.EntitlementType) db_entitlement.EntitlementType {
					return db_entitlement.EntitlementType(t)
				})...))
			}

			if len(params.IDs) > 0 {
				query = query.Where(db_entitlement.IDIn(params.IDs...))
			}

			if len(params.FeatureIDsOrKeys) > 0 {
				var ep predicate.Entitlement
				for i, idOrKey := range params.FeatureIDsOrKeys {
					p := db_entitlement.Or(db_entitlement.FeatureID(idOrKey), db_entitlement.FeatureKey(idOrKey))
					if i == 0 {
						ep = p
						continue
					}
					ep = db_entitlement.Or(ep, p)
				}
				query = query.Where(ep)
			}

			if len(params.FeatureIDs) > 0 {
				query = query.Where(db_entitlement.FeatureIDIn(params.FeatureIDs...))
			}

			if len(params.FeatureKeys) > 0 {
				query = query.Where(db_entitlement.FeatureKeyIn(params.FeatureKeys...))
			}

			if !params.IncludeDeleted {
				query = query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(now), db_entitlement.DeletedAtIsNil()))
			}

			if !params.IncludeDeletedAfter.IsZero() {
				query = query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(params.IncludeDeletedAfter), db_entitlement.DeletedAtIsNil()))
			}

			if params.ExcludeInactive {
				query = query.Where(EntitlementActiveAt(now)...)
			}

			if params.ActiveAt != nil {
				query = query.Where(EntitlementActiveAt(*params.ActiveAt)...)
			}

			if params.OrderBy != "" {
				order := []sql.OrderTermOption{}
				if !params.Order.IsDefaultValue() {
					order = entutils.GetOrdering(params.Order)
				}
				switch params.OrderBy {
				case entitlement.ListEntitlementsOrderByCreatedAt:
					query = query.Order(db_entitlement.ByCreatedAt(order...))
				case entitlement.ListEntitlementsOrderByUpdatedAt:
					query = query.Order(db_entitlement.ByUpdatedAt(order...))
				}
			}

			response := pagination.Result[entitlement.Entitlement]{
				Page: params.Page,
			}

			// we're using limit and offset
			if params.Page.IsZero() {
				if params.Limit > 0 {
					query = query.Limit(params.Limit)
				}
				if params.Offset > 0 {
					query = query.Offset(params.Offset)
				}

				entities, err := query.All(ctx)
				if err != nil {
					return response, err
				}

				mapped := make([]entitlement.Entitlement, 0, len(entities))
				for _, entity := range entities {
					mappedEnt, err := repo.mapEntitlementEntity(entity)
					if err != nil {
						return response, err
					}
					mapped = append(mapped, *mappedEnt)
				}

				response.Items = mapped
				response.TotalCount = len(mapped)
				return response, nil
			}

			paged, err := query.Paginate(ctx, params.Page)
			if err != nil {
				return response, err
			}

			result := make([]entitlement.Entitlement, 0, len(paged.Items))
			for _, e := range paged.Items {
				mapped, err := repo.mapEntitlementEntity(e)
				if err != nil {
					return response, err
				}
				result = append(result, *mapped)
			}

			response.TotalCount = paged.TotalCount
			response.Items = result

			return response, nil
		},
	)
}

func (a *entitlementDBAdapter) mapEntitlementEntity(e *db.Entitlement) (*entitlement.Entitlement, error) {
	if e.Edges.Customer == nil {
		return nil, fmt.Errorf("entitlement %s has no customer", e.ID)
	}

	ent := &entitlement.Entitlement{
		GenericProperties: entitlement.GenericProperties{
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: e.CreatedAt.UTC(),
				UpdatedAt: e.UpdatedAt.UTC(),
				DeletedAt: convert.SafeToUTC(e.DeletedAt),
			},
			MetadataModel: models.MetadataModel{
				Metadata: e.Metadata,
			},
			Annotations:     e.Annotations,
			ID:              e.ID,
			FeatureID:       e.FeatureID,
			FeatureKey:      e.FeatureKey,
			EntitlementType: entitlement.EntitlementType(e.EntitlementType),
			ActiveFrom:      convert.SafeToUTC(e.ActiveFrom),
			ActiveTo:        convert.SafeToUTC(e.ActiveTo),
		},
		MeasureUsageFrom:        e.MeasureUsageFrom,
		IssueAfterReset:         e.IssueAfterReset,
		IssueAfterResetPriority: e.IssueAfterResetPriority,
		IsSoftLimit:             e.IsSoftLimit,
		PreserveOverageAtReset:  e.PreserveOverageAtReset,
	}

	if mapped, mapErr := customeradapter.CustomerFromDBEntity(*e.Edges.Customer, customer.Expands{}); mapErr == nil {
		ent.Customer = mapped
	}

	switch {
	case len(e.Edges.UsageReset) > 0:
		ent.LastReset = convert.ToPointer(e.Edges.UsageReset[0].ResetTime.In(time.UTC))
	case e.MeasureUsageFrom != nil:
		ent.LastReset = convert.ToPointer(e.MeasureUsageFrom.In(time.UTC))
	}

	if e.Config != nil {
		ent.Config = e.Config
	}

	if e.UsagePeriodAnchor != nil && e.UsagePeriodInterval != nil {
		var inps []entitlement.UsagePeriodInput

		parsed, err := e.UsagePeriodInterval.Parse()
		if err != nil {
			return nil, err
		}

		// Let's add the initial
		inps = append(inps, timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
			if e.MeasureUsageFrom != nil {
				return e.MeasureUsageFrom.In(time.UTC)
			}

			return e.UsagePeriodAnchor.In(time.UTC)
		})(timeutil.Recurrence{
			Anchor:   e.UsagePeriodAnchor.In(time.UTC),
			Interval: timeutil.RecurrenceInterval{ISODuration: parsed},
		}))

		ent.OriginalUsagePeriodAnchor = convert.SafeToUTC(e.UsagePeriodAnchor)

		// We no longer override the anchor at each reset as we need to preserve the original
		// We populate in reverse order (last = oldest first)
		for i := len(e.Edges.UsageReset) - 1; i >= 0; i-- {
			reset := e.Edges.UsageReset[i]
			if reset == nil {
				continue
			}

			parsed, err := reset.UsagePeriodInterval.Parse()
			if err != nil {
				return nil, err
			}

			inps = append(inps, timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
				return reset.ResetTime.In(time.UTC)
			})(timeutil.Recurrence{
				Anchor:   reset.Anchor.In(time.UTC),
				Interval: timeutil.RecurrenceInterval{ISODuration: parsed},
			}))
		}

		ent.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriod(inps))
	}

	if e.CurrentUsagePeriodEnd != nil && e.CurrentUsagePeriodStart != nil {
		ent.CurrentUsagePeriod = &timeutil.ClosedPeriod{
			From: e.CurrentUsagePeriodStart.In(time.UTC),
			To:   e.CurrentUsagePeriodEnd.In(time.UTC),
		}
	}

	// Let's update the current usage period
	if ent.UsagePeriod != nil {
		cp, err := ent.UsagePeriod.GetCurrentPeriodAt(clock.Now())
		if err == nil {
			ent.CurrentUsagePeriod = &cp
		}
	}

	return ent, nil
}

func (a *entitlementDBAdapter) UpdateEntitlementUsagePeriod(ctx context.Context, entitlementID models.NamespacedID, params entitlement.UpdateEntitlementUsagePeriodParams) error {
	_, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
			update := repo.db.Entitlement.Update().
				Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
				SetCurrentUsagePeriodStart(params.CurrentUsagePeriod.From).
				SetCurrentUsagePeriodEnd(params.CurrentUsagePeriod.To)

			_, err := update.Save(ctx)
			return nil, err
		},
	)
	return err
}

func (a *entitlementDBAdapter) UpsertEntitlementCurrentPeriods(ctx context.Context, updates []entitlement.UpsertEntitlementCurrentPeriodElement) error {
	return entutils.TransactingRepoWithNoValue(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) error {
			// Let's make sure there aren't any duplicate updates
			uniqueUpdates := lo.UniqBy(updates, func(u entitlement.UpsertEntitlementCurrentPeriodElement) string {
				return fmt.Sprintf("%s:%s", u.ID, u.Namespace)
			})

			if len(uniqueUpdates) != len(updates) {
				return fmt.Errorf("%d duplicate entitlement updates provided", len(updates)-len(uniqueUpdates))
			}

			now := clock.Now()

			// We will check that all provided entitlements exist as we don't want to create any, just update the current usage period
			entitlements, err := repo.db.Entitlement.Query().
				// We're ignoring namespace here but as IDs are globally unique this should be fine
				Where(db_entitlement.IDIn(slicesx.Map(updates, func(u entitlement.UpsertEntitlementCurrentPeriodElement) string {
					return u.ID
				})...),
				).
				WithCustomer(
					func(q *db.CustomerQuery) {
						customeradapter.WithSubjects(q, now)
					},
				).
				All(ctx)
			if err != nil {
				return err
			}

			if len(entitlements) != len(updates) {
				return fmt.Errorf("%d entitlement updates provided do not exist", len(updates)-len(entitlements))
			}

			// Create a map of entitlements for faster lookup
			entMap := make(map[string]*db.Entitlement)
			for _, ent := range entitlements {
				entMap[ent.ID] = ent
			}

			// Now we will proceed with the update
			dbUpdates := make([]*db.EntitlementCreate, 0, len(updates))
			for _, update := range updates {
				ent, ok := entMap[update.ID]
				if !ok {
					return fmt.Errorf("inconsistency error: entitlement %s not found", update.ID)
				}

				create := repo.db.Entitlement.Create().
					// These fields will be ignored in the custom update
					SetID(update.ID).
					SetNamespace(ent.Namespace).
					SetEntitlementType(ent.EntitlementType).
					SetFeatureID(ent.FeatureID).
					SetFeatureKey(ent.FeatureKey).
					SetCustomerID(ent.CustomerID).
					SetCurrentUsagePeriodStart(update.CurrentUsagePeriod.From).
					SetCurrentUsagePeriodEnd(update.CurrentUsagePeriod.To)

				dbUpdates = append(dbUpdates, create)
			}

			// Let's try to come up with a sensible limiting to avoid hitting PG's limit on max number of parameters
			// We'll assume each upsert contributing len(Columns) parameters
			// Let's also subtract 4 for the ON CONFLICT columns
			dbUpdatesChunks := lo.Chunk(dbUpdates, (MAX_POSTGRES_QUERY_PARAMS-4)/len(db_entitlement.Columns))

			// Let's preserve the atomic nature of the operation by running inside a transaction
			return transaction.RunWithNoValue(ctx, repo, func(ctx context.Context) error {
				for _, chunk := range dbUpdatesChunks {
					// Let's do a batch insert with on conflict do update
					// Let's do a batch insert with on conflict do update
					err = repo.db.Entitlement.CreateBulk(chunk...).
						OnConflict(
							sql.ConflictColumns(db_entitlement.FieldID),
							sql.ResolveWith(func(u *sql.UpdateSet) {
								u.SetExcluded(db_entitlement.FieldCurrentUsagePeriodStart).
									SetExcluded(db_entitlement.FieldCurrentUsagePeriodEnd)
							})).Exec(ctx)
					if err != nil {
						return err
					}
				}

				return nil
			})
		},
	)
}

func (a *entitlementDBAdapter) ListActiveEntitlementsWithExpiredUsagePeriod(ctx context.Context, params entitlement.ListExpiredEntitlementsParams) ([]entitlement.Entitlement, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]entitlement.Entitlement, error) {
			now := clock.Now()

			query := withAllUsageResets(repo.db.Entitlement.Query(), params.Namespaces).
				WithCustomer(func(q *db.CustomerQuery) {
					customeradapter.WithSubjects(q, now)
				}).
				Where(EntitlementActiveAt(params.Highwatermark)...).
				Where(
					db_entitlement.CurrentUsagePeriodEndNotNil(),
					db_entitlement.CurrentUsagePeriodEndLTE(params.Highwatermark),
					db_entitlement.Or(db_entitlement.DeletedAtIsNil(), db_entitlement.DeletedAtGT(now)),
				)

			if len(params.Namespaces) > 0 {
				query = query.Where(db_entitlement.NamespaceIn(params.Namespaces...))
			}

			// Let's order by cursor
			query = query.Order(
				db_entitlement.ByCreatedAt(sql.OrderAsc()),
				db_entitlement.ByID(sql.OrderAsc()),
			)

			// Let's handle cursoring for the given order
			if params.Cursor != nil {
				query = query.Where(
					db_entitlement.Or(
						db_entitlement.CreatedAtGT(params.Cursor.Time),
						db_entitlement.And(
							db_entitlement.CreatedAt(params.Cursor.Time),
							db_entitlement.IDGT(params.Cursor.ID),
						),
					),
				)
			}

			// Let's handle limit
			if params.Limit != 0 {
				query = query.Limit(params.Limit)
			}

			res, err := query.All(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]entitlement.Entitlement, 0, len(res))
			for _, e := range res {
				mapped, err := repo.mapEntitlementEntity(e)
				if err != nil {
					return nil, err
				}

				// Let's set back the original current usage period
				if e.CurrentUsagePeriodStart != nil && e.CurrentUsagePeriodEnd != nil {
					mapped.CurrentUsagePeriod = &timeutil.ClosedPeriod{
						From: e.CurrentUsagePeriodStart.In(time.UTC),
						To:   e.CurrentUsagePeriodEnd.In(time.UTC),
					}
				}

				result = append(result, *mapped)
			}

			return result, nil
		},
	)
}

func (a *entitlementDBAdapter) LockEntitlementForTx(ctx context.Context, tx *entutils.TxDriver, entitlementID models.NamespacedID) error {
	pgLockNotAvailableErrorCode := "55P03"

	if tx == nil {
		return fmt.Errorf("lock entitlement for tx called from outside a transaction")
	}
	_, err := a.WithTx(ctx, tx).db.Entitlement.
		Query().
		Where(db_entitlement.ID(entitlementID.ID), db_entitlement.Namespace(entitlementID.Namespace)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		if strings.Contains(err.Error(), pgLockNotAvailableErrorCode) {
			// TODO: return a more specific error
			return fmt.Errorf("acquiring lock for entitlement %s failed: %w", entitlementID.ID, err)
		}
	}

	return err
}

type namespacesWithCount struct {
	Namespace string
	Count     int
}

func (a *entitlementDBAdapter) ListNamespacesWithActiveEntitlements(ctx context.Context, includeDeletedAfter time.Time) ([]string, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) ([]string, error) {
			now := clock.Now()
			namespaces := []namespacesWithCount{}

			query := repo.db.Entitlement.Query().
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtGT(includeDeletedAfter), db_entitlement.DeletedAtIsNil()),
				)

			if includeDeletedAfter.Before(now) || includeDeletedAfter.Equal(now) {
				query = query.Where(EntitlementActiveAt(now)...)
			} else {
				query = query.Where(entitlementActiveBetween(includeDeletedAfter, now)...)
			}

			query2 := query.
				GroupBy(db_entitlement.FieldNamespace).
				Aggregate(db.Count())

			err := query2.Scan(ctx, &namespaces)
			if err != nil {
				return nil, err
			}

			return slicesx.Map(namespaces, func(n namespacesWithCount) string {
				return n.Namespace
			}), nil
		},
	)
}

func (a *entitlementDBAdapter) GetScheduledEntitlements(ctx context.Context, namespace string, customerID string, featureKey string, starting time.Time) ([]entitlement.Entitlement, error) {
	res, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *entitlementDBAdapter) (*[]entitlement.Entitlement, error) {
			now := clock.Now()

			query := repo.db.Entitlement.Query().WithCustomer(func(q *db.CustomerQuery) {
				customeradapter.WithSubjects(q, now)
			})
			query = withAllUsageResets(query, []string{namespace})
			res, err := query.
				Where(
					db_entitlement.Or(
						db_entitlement.ActiveToIsNil(),
						db_entitlement.ActiveToGT(starting),
					),
				).
				Where(
					db_entitlement.Or(db_entitlement.DeletedAtIsNil(), db_entitlement.DeletedAtGT(now)),
					db_entitlement.Namespace(namespace),
					db_entitlement.HasCustomerWith(
						customerdb.Namespace(namespace),
						customerNotDeletedAt(now),
						customerdb.ID(customerID),
					),
					db_entitlement.FeatureKey(featureKey),
				).Order(
				func(s *sql.Selector) {
					// order by COALESCE(ActiveFrom, CreatedAt) ASC
					orderBy := fmt.Sprintf("COALESCE(%s, %s) ASC", db_entitlement.FieldActiveFrom, db_entitlement.FieldCreatedAt)
					s.OrderBy(orderBy)
				},
			).All(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]entitlement.Entitlement, 0, len(res))
			for _, e := range res {
				mapped, err := repo.mapEntitlementEntity(e)
				if err != nil {
					return nil, err
				}
				result = append(result, *mapped)
			}

			return &result, nil
		},
	)
	return defaultx.WithDefault(res, nil), err
}

func entitlementActiveBetween(from, to time.Time) []predicate.Entitlement {
	return []predicate.Entitlement{
		db_entitlement.Or(
			db_entitlement.And(db_entitlement.ActiveFromIsNil(), db_entitlement.CreatedAtLTE(to)),
			db_entitlement.ActiveFromLTE(to),
		),
		db_entitlement.Or(
			db_entitlement.ActiveToIsNil(),
			db_entitlement.ActiveToGT(from),
		),
	}
}

func customerNotDeletedAt(at time.Time) predicate.Customer {
	return customerdb.Or(
		customerdb.DeletedAtGT(at),
		customerdb.DeletedAtIsNil(),
	)
}

// EntitlementActiveAt is exposed to be used for subscription adapter
func EntitlementActiveAt(at time.Time) []predicate.Entitlement {
	return []predicate.Entitlement{
		db_entitlement.Or(
			// If activeFrom is nil activity starts at creation time
			db_entitlement.And(db_entitlement.ActiveFromIsNil(), db_entitlement.CreatedAtLTE(at)),
			db_entitlement.ActiveFromLTE(at),
		),
		db_entitlement.Or(
			db_entitlement.ActiveToIsNil(),
			db_entitlement.ActiveToGT(at),
		),
	}
}

func withAllUsageResets(q *db.EntitlementQuery, namespaces []string) *db.EntitlementQuery {
	return q.WithUsageReset(func(urq *db.UsageResetQuery) {
		urq.
			Order(db_usagereset.ByResetTime(sql.OrderDesc()))

		if len(namespaces) > 0 {
			urq.Where(db_usagereset.NamespaceIn(namespaces...))
		}
	})
}

const (
	MAX_POSTGRES_QUERY_PARAMS = 65535
)
