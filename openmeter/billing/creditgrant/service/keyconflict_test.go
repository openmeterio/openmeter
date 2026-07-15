package service

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// stubCreditPurchaseService fakes only List; the embedded interface panics on
// anything else, which keeps the fake honest about what enrichKeyConflict uses.
type stubCreditPurchaseService struct {
	creditpurchase.Service

	listResult pagination.Result[creditpurchase.Charge]
	listErr    error
	gotList    creditpurchase.ListChargesInput
}

func (s *stubCreditPurchaseService) List(_ context.Context, input creditpurchase.ListChargesInput) (pagination.Result[creditpurchase.Charge], error) {
	s.gotList = input
	return s.listResult, s.listErr
}

func TestEnrichKeyConflict(t *testing.T) {
	newConflict := func() *creditpurchase.ChargeKeyConflictError {
		err := creditpurchase.NewChargeKeyConflictError("ns-1", "customer-1", "key-1")

		var conflict *creditpurchase.ChargeKeyConflictError
		require.True(t, errors.As(err, &conflict))

		return conflict
	}

	existingCharge := func(id string) creditpurchase.Charge {
		charge := creditpurchase.Charge{}
		charge.ID = id
		return charge
	}

	t.Run("a single match attaches the existing grant as the conflicting resource", func(t *testing.T) {
		stub := &stubCreditPurchaseService{
			listResult: pagination.Result[creditpurchase.Charge]{
				Items:      []creditpurchase.Charge{existingCharge("grant-1")},
				TotalCount: 1,
			},
		}
		svc := &service{creditPurchaseService: stub}

		err := svc.enrichKeyConflict(t.Context(), newConflict())

		require.True(t, models.IsGenericConflictError(err))

		conflict, ok := lo.ErrorsAs[*models.GenericConflictError](err)
		require.True(t, ok)
		require.NotNil(t, conflict.Resource())
		require.Equal(t, models.ConflictingResource{
			Type:       "credit_grant",
			ID:         "grant-1",
			CustomerID: "customer-1",
		}, *conflict.Resource())

		require.Equal(t, "ns-1", stub.gotList.Namespace)
		require.Equal(t, []string{"customer-1"}, stub.gotList.CustomerIDs)
		require.NotNil(t, stub.gotList.Key)
		require.Equal(t, lo.ToPtr("key-1"), stub.gotList.Key.Eq)
	})

	t.Run("no match degrades to the plain conflict", func(t *testing.T) {
		stub := &stubCreditPurchaseService{}
		svc := &service{creditPurchaseService: stub}

		err := svc.enrichKeyConflict(t.Context(), newConflict())

		require.True(t, models.IsGenericConflictError(err))

		conflict, ok := lo.ErrorsAs[*models.GenericConflictError](err)
		require.True(t, ok)
		require.Nil(t, conflict.Resource())
	})

	t.Run("a lookup failure degrades to the plain conflict", func(t *testing.T) {
		stub := &stubCreditPurchaseService{listErr: errors.New("db down")}
		svc := &service{creditPurchaseService: stub}

		err := svc.enrichKeyConflict(t.Context(), newConflict())

		require.True(t, models.IsGenericConflictError(err))

		conflict, ok := lo.ErrorsAs[*models.GenericConflictError](err)
		require.True(t, ok)
		require.Nil(t, conflict.Resource())
	})
}
