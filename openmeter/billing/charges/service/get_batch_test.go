package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/models"
)

func managedResource(namespace, id string) meta.ManagedResource {
	return meta.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: namespace}, ID: id}
}

type batchRecordingUsageBased struct {
	usagebased.Service
	batches [][]string
}

func (f *batchRecordingUsageBased) GetByIDs(_ context.Context, input usagebased.GetByIDsInput) ([]usagebased.Charge, error) {
	f.batches = append(f.batches, input.IDs)
	return lo.Map(input.IDs, func(id string, _ int) usagebased.Charge {
		return usagebased.Charge{ChargeBase: usagebased.ChargeBase{ManagedResource: managedResource(input.Namespace, id)}}
	}), nil
}

type batchRecordingFlatFee struct {
	flatfee.Service
	batches [][]string
}

func (f *batchRecordingFlatFee) GetByIDs(_ context.Context, input flatfee.GetByIDsInput) ([]flatfee.Charge, error) {
	f.batches = append(f.batches, input.IDs)
	return lo.Map(input.IDs, func(id string, _ int) flatfee.Charge {
		return flatfee.Charge{ChargeBase: flatfee.ChargeBase{ManagedResource: managedResource(input.Namespace, id)}}
	}), nil
}

type batchRecordingCreditPurchase struct {
	creditpurchase.Service
	batches [][]string
}

func (f *batchRecordingCreditPurchase) GetByIDs(_ context.Context, input creditpurchase.GetByIDsInput) ([]creditpurchase.Charge, error) {
	f.batches = append(f.batches, input.IDs)
	return lo.Map(input.IDs, func(id string, _ int) creditpurchase.Charge {
		return creditpurchase.Charge{ChargeBase: creditpurchase.ChargeBase{ManagedResource: managedResource(input.Namespace, id)}}
	}), nil
}

func TestExpandChargesWithTypesLoadsInBatches(t *testing.T) {
	const namespace = "ns"

	// given more charges of one type than fit in a batch, interleaved with
	// charges of the other types
	var items charges.ChargeSearchItems
	for i := range 2*chargeLoadBatchSize + 500 {
		items = append(items, charges.ChargeSearchItem{
			ID:   meta.ChargeID{Namespace: namespace, ID: fmt.Sprintf("ub-%d", i)},
			Type: meta.ChargeTypeUsageBased,
		})
		if i%10 == 0 {
			items = append(items, charges.ChargeSearchItem{
				ID:   meta.ChargeID{Namespace: namespace, ID: fmt.Sprintf("ff-%d", i)},
				Type: meta.ChargeTypeFlatFee,
			})
		}
	}
	items = append(items, charges.ChargeSearchItem{
		ID:   meta.ChargeID{Namespace: namespace, ID: "cp-0"},
		Type: meta.ChargeTypeCreditPurchase,
	})

	usageBased := &batchRecordingUsageBased{}
	flatFees := &batchRecordingFlatFee{}
	creditPurchases := &batchRecordingCreditPurchase{}
	s := &service{usageBasedService: usageBased, flatFeeService: flatFees, creditPurchaseService: creditPurchases}

	// when the charges are expanded
	out, err := s.expandChargesWithTypes(t.Context(), namespace, items, nil)
	require.NoError(t, err)

	// then no load gets more than a batch, every ID is loaded once, and the
	// search order is kept
	var loaded []string
	for _, batches := range [][][]string{usageBased.batches, flatFees.batches, creditPurchases.batches} {
		for _, batch := range batches {
			require.LessOrEqual(t, len(batch), chargeLoadBatchSize)
			loaded = append(loaded, batch...)
		}
	}
	require.Len(t, usageBased.batches, 3)

	wantIDs := lo.Map(items, func(item charges.ChargeSearchItem, _ int) string { return item.ID.ID })
	require.ElementsMatch(t, wantIDs, loaded)
	require.Equal(t, wantIDs, lo.Map(out, func(charge charges.Charge, _ int) string { return charge.GetID() }))
}
