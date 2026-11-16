package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// stubTaxCodeResolver serves BatchResolve from taxCodes, or fails with err, and records the calls.
// Calling anything else panics on the nil embedded interface.
type stubTaxCodeResolver struct {
	productcatalog.TransactionalTaxCodeResolver

	taxCodes map[string]*taxcode.TaxCode
	err      error
	calls    int
}

func (r *stubTaxCodeResolver) BatchResolve(_ context.Context, _ string, ids ...string) (map[string]*taxcode.TaxCode, error) {
	r.calls++

	if r.err != nil {
		return nil, r.err
	}

	return lo.PickByKeys(r.taxCodes, ids), nil
}

// ValidateTaxCodeID tells a deleted Tax Code apart from a missing one with the shared billing
// codes, and never reports a failed lookup as a validation error.
func TestValidateTaxCodeID(t *testing.T) {
	const (
		namespace = "ns"
		liveID    = "01JTAXCODE00000000000000000"
		deletedID = "01JTAXCODE00000000000000001"
		missingID = "01JTAXCODE00000000000000002"
	)

	resolver := &stubTaxCodeResolver{taxCodes: map[string]*taxcode.TaxCode{
		liveID: {NamespacedID: models.NamespacedID{Namespace: namespace, ID: liveID}},
		deletedID: {
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deletedID},
			ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(clock.Now().Add(-1))},
		},
	}}

	t.Run("live tax code passes", func(t *testing.T) {
		require.NoError(t, ValidateTaxCodeID(t.Context(), resolver, namespace, liveID))
	})

	t.Run("empty id passes without a lookup", func(t *testing.T) {
		calls := resolver.calls

		require.NoError(t, ValidateTaxCodeID(t.Context(), resolver, namespace, ""))
		assert.Equal(t, calls, resolver.calls)
	})

	t.Run("deleted tax code is rejected as deleted", func(t *testing.T) {
		err := ValidateTaxCodeID(t.Context(), resolver, namespace, deletedID)

		require.ErrorIs(t, err, ErrTaxCodeDeleted)
		assert.ErrorContains(t, err, deletedID)
	})

	t.Run("missing tax code is rejected as not found", func(t *testing.T) {
		err := ValidateTaxCodeID(t.Context(), resolver, namespace, missingID)

		require.ErrorIs(t, err, ErrTaxCodeNotFound)
		assert.ErrorContains(t, err, missingID)
	})

	t.Run("lookup failure is not a validation error", func(t *testing.T) {
		lookupErr := errors.New("db down")

		err := ValidateTaxCodeID(t.Context(), &stubTaxCodeResolver{err: lookupErr}, namespace, liveID)

		require.ErrorIs(t, err, lookupErr)
		var issue ValidationIssue
		assert.False(t, errors.As(err, &issue), "lookup failures must not become validation issues")
	})
}
