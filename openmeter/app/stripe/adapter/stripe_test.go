package appstripeadapter

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
)

func TestDeleteStripeAppDataSoftDeletesAndSanitizesSecrets(t *testing.T) {
	// Given an active Stripe subtype containing provider metadata and both secret references.
	// When the subtype is deleted.
	// Then the row and its metadata remain while its credentials become inaccessible.
	env := newStripeCustomerTestEnv(t)
	appID := env.createStripeApp(t, ulid.Make().String())

	before, err := env.db.AppStripe.Get(t.Context(), appID.ID)
	require.NoError(t, err)
	require.NotNil(t, before.APIKey)
	require.NotNil(t, before.WebhookSecret)

	err = env.adapter.DeleteStripeAppData(t.Context(), appstripe.DeleteStripeAppDataInput{AppID: appID})
	require.NoError(t, err)

	after, err := env.db.AppStripe.Get(t.Context(), appID.ID)
	require.NoError(t, err)
	require.NotNil(t, after.DeletedAt)
	require.Nil(t, after.APIKey)
	require.Nil(t, after.WebhookSecret)
	require.Equal(t, before.StripeAccountID, after.StripeAccountID)
	require.Equal(t, before.StripeLivemode, after.StripeLivemode)
	require.Equal(t, before.MaskedAPIKey, after.MaskedAPIKey)
	require.Equal(t, before.StripeWebhookID, after.StripeWebhookID)

	_, err = env.adapter.GetStripeAppData(t.Context(), appstripe.GetStripeAppDataInput{AppID: appID})
	require.ErrorIs(t, err, app.ErrAppDeleted)

	deletedData, err := env.adapter.GetStripeAppData(t.Context(), appstripe.GetStripeAppDataInput{
		AppID:          appID,
		IncludeDeleted: true,
	})
	require.NoError(t, err)
	require.Equal(t, before.StripeAccountID, deletedData.StripeAccountID)
	require.Equal(t, before.StripeLivemode, deletedData.Livemode)
	require.Equal(t, before.MaskedAPIKey, deletedData.MaskedAPIKey)
	require.Equal(t, before.StripeWebhookID, deletedData.StripeWebhookID)
	require.Empty(t, deletedData.APIKey.ID)
	require.Empty(t, deletedData.WebhookSecret.ID)

	_, err = env.adapter.GetWebhookSecret(t.Context(), appstripe.GetWebhookSecretInput{AppID: appID.ID})
	require.True(t, app.IsAppNotFoundError(err))
}

func TestStripeAppSecretLifecycleConstraint(t *testing.T) {
	// Given an active Stripe subtype with both required secret references.
	// When only part of the credential lifecycle transition is persisted.
	// Then the database rejects the inconsistent state.
	env := newStripeCustomerTestEnv(t)
	appID := env.createStripeApp(t, ulid.Make().String())

	t.Run("active row cannot lose only the API key", func(t *testing.T) {
		_, err := env.db.AppStripe.UpdateOneID(appID.ID).
			ClearAPIKey().
			Save(t.Context())
		require.Error(t, err)
	})

	t.Run("row cannot be marked deleted while retaining secrets", func(t *testing.T) {
		_, err := env.db.AppStripe.UpdateOneID(appID.ID).
			SetDeletedAt(time.Now()).
			Save(t.Context())
		require.Error(t, err)
	})
}
