package appstripeadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	appstripeinvoicegenerationdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripeinvoicegeneration"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestInvoiceGeneration(t *testing.T) {
	env := newStripeCustomerTestEnv(t)

	invoiceID := billing.InvoiceID{Namespace: ulid.Make().String(), ID: ulid.Make().String()}
	otherInvoiceID := billing.InvoiceID{Namespace: invoiceID.Namespace, ID: ulid.Make().String()}

	t.Run("an invoice without a generation is at 0", func(t *testing.T) {
		generation, err := env.adapter.GetLatestInvoiceGeneration(t.Context(), invoiceID)
		require.NoError(t, err)
		require.Equal(t, 0, generation)
	})

	t.Run("every increment adds one for that invoice only", func(t *testing.T) {
		require.NoError(t, env.adapter.IncrementInvoiceGeneration(t.Context(), invoiceID))
		require.NoError(t, env.adapter.IncrementInvoiceGeneration(t.Context(), invoiceID))

		generation, err := env.adapter.GetLatestInvoiceGeneration(t.Context(), invoiceID)
		require.NoError(t, err)
		require.Equal(t, 2, generation)

		generation, err = env.adapter.GetLatestInvoiceGeneration(t.Context(), otherInvoiceID)
		require.NoError(t, err)
		require.Equal(t, 0, generation)
	})

	t.Run("an increment moves updated_at and keeps created_at", func(t *testing.T) {
		// given: an invoice whose generation was incremented once
		timedInvoiceID := billing.InvoiceID{Namespace: invoiceID.Namespace, ID: ulid.Make().String()}
		createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

		clock.FreezeTime(createdAt)
		defer clock.UnFreeze()

		require.NoError(t, env.adapter.IncrementInvoiceGeneration(t.Context(), timedInvoiceID))

		// when: it is incremented again an hour later
		updatedAt := createdAt.Add(time.Hour)
		clock.FreezeTime(updatedAt)

		require.NoError(t, env.adapter.IncrementInvoiceGeneration(t.Context(), timedInvoiceID))

		// then: only updated_at follows the second increment
		row, err := env.db.AppStripeInvoiceGeneration.Query().
			Where(appstripeinvoicegenerationdb.InvoiceID(timedInvoiceID.ID)).
			Only(t.Context())
		require.NoError(t, err)
		require.Equal(t, 2, row.Generation)
		require.True(t, createdAt.Equal(row.CreatedAt))
		require.True(t, updatedAt.Equal(row.UpdatedAt))
		require.Nil(t, row.DeletedAt)
	})

	t.Run("an increment survives the rollback of the surrounding transaction", func(t *testing.T) {
		// given: an invoice sync running in a transaction
		errSyncFailed := errors.New("invoice sync failed")

		// when: the generation is incremented and the transaction is rolled back
		err := transaction.RunWithNoValue(t.Context(), env.adapter, func(ctx context.Context) error {
			if err := env.adapter.IncrementInvoiceGeneration(ctx, otherInvoiceID); err != nil {
				return err
			}

			return errSyncFailed
		})
		require.ErrorIs(t, err, errSyncFailed)

		// then: the next create request still gets a new idempotency key
		generation, err := env.adapter.GetLatestInvoiceGeneration(t.Context(), otherInvoiceID)
		require.NoError(t, err)
		require.Equal(t, 1, generation)
	})
}
