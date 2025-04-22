package adapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/app"
	custominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicing"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ custominvoicing.AppConfigAdapter = (*adapter)(nil)

func (a *adapter) GetAppConfiguration(ctx context.Context, input app.AppID) (custominvoicing.Configuration, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (custominvoicing.Configuration, error) {
		appConfig, err := tx.db.AppCustomInvoicing.Query().
			Where(
				appcustominvoicing.ID(input.ID),
				appcustominvoicing.Namespace(input.Namespace),
				appcustominvoicing.DeletedAtIsNil(),
			).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return custominvoicing.Configuration{}, nil
			}

			return custominvoicing.Configuration{}, err
		}

		return mapDBToAppConfiguration(appConfig), nil
	})
}

func (a *adapter) UpsertAppConfiguration(ctx context.Context, input custominvoicing.UpsertAppConfigurationInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.AppCustomInvoicing.Create().
			SetID(input.AppID.ID).
			SetNamespace(input.AppID.Namespace).
			SetEnableDraftSyncHook(input.Configuration.EnableDraftSyncHook).
			SetEnableIssuingSyncHook(input.Configuration.EnableIssuingSyncHook).

			// Upsert
			OnConflictColumns(appcustominvoicing.FieldID, appcustominvoicing.FieldNamespace).
			UpdateNewValues().
			Exec(ctx)
	})
}

func (a *adapter) DeleteAppConfiguration(ctx context.Context, input app.AppID) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.AppCustomInvoicing.Update().
			Where(
				appcustominvoicing.ID(input.ID),
				appcustominvoicing.Namespace(input.Namespace),
				appcustominvoicing.DeletedAtIsNil(),
			).
			SetDeletedAt(time.Now()).
			Exec(ctx)
	})
}

func mapDBToAppConfiguration(appConfig *db.AppCustomInvoicing) custominvoicing.Configuration {
	return custominvoicing.Configuration{
		EnableDraftSyncHook:   appConfig.EnableDraftSyncHook,
		EnableIssuingSyncHook: appConfig.EnableIssuingSyncHook,
	}
}
