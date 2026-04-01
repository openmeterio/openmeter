package creditrealization

import (
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// testCurrency returns a USD calculator for tests.
func testCurrency(t *testing.T) currencyx.Calculator {
	t.Helper()

	calc, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

	return calc
}

var testServicePeriod = timeutil.ClosedPeriod{
	From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	To:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
}

// allocationBuilder builds Realization entries of type allocation for tests.
type allocationBuilder struct {
	createdAt time.Time
	sortHint  int
}

func newAllocationBuilder() *allocationBuilder {
	return &allocationBuilder{
		createdAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (b *allocationBuilder) withCreatedAt(t time.Time) *allocationBuilder {
	b.createdAt = t
	return b
}

func (b *allocationBuilder) build(amount float64) Realization {
	id := uuid.New().String()
	b.sortHint++

	return Realization{
		NamespacedModel: models.NamespacedModel{
			Namespace: "test-ns",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: b.createdAt,
			UpdatedAt: b.createdAt,
		},
		CreateInput: CreateInput{
			ID:            id,
			ServicePeriod: testServicePeriod,
			Amount:        alpacadecimal.NewFromFloat(amount),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: uuid.New().String(),
			},
			Type: TypeAllocation,
		},
		SortHint: b.sortHint,
	}
}

// correctionFor builds a correction Realization targeting the given allocation.
func correctionFor(allocation Realization, amount float64) Realization {
	return Realization{
		NamespacedModel: models.NamespacedModel{
			Namespace: "test-ns",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		CreateInput: CreateInput{
			ID:            uuid.New().String(),
			ServicePeriod: allocation.ServicePeriod,
			Amount:        alpacadecimal.NewFromFloat(amount),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: uuid.New().String(),
			},
			Type:                  TypeCorrection,
			CorrectsRealizationID: lo.ToPtr(allocation.ID),
		},
		SortHint: 0,
	}
}

// correctionInputsSum returns the total amount across all correction inputs.
func correctionInputsSum(inputs CreateCorrectionInputs) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, input := range inputs {
		sum = sum.Add(input.Amount.Abs())
	}
	return sum
}

// correctionRequestAmounts extracts the correction amounts as float64 slice for easy assertion.
func correctionRequestAmounts(cr CorrectionRequest) []float64 {
	out := make([]float64, len(cr))
	for i, item := range cr {
		out[i] = item.Amount.InexactFloat64()
	}
	return out
}

// correctionRequestAllocationIDs extracts the allocation IDs from a correction request.
func correctionRequestAllocationIDs(cr CorrectionRequest) []string {
	out := make([]string, len(cr))
	for i, item := range cr {
		out[i] = item.Allocation.ID
	}
	return out
}

func TestCreateCorrectionRequest(t *testing.T) {
	t.Run("full revert of single allocation", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		cr, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-10),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, alloc.ID, cr[0].Allocation.ID)
		assert.Equal(t, -10.0, cr[0].Amount.InexactFloat64())
	})

	t.Run("partial revert of single allocation", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		cr, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-3),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, -3.0, cr[0].Amount.InexactFloat64())
	})

	t.Run("full revert spanning multiple allocations in reverse order", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		a3 := b.build(2)

		cr, err := Realizations{a1, a2, a3}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-10),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 3)
		// Reverse order: a3, a2, a1
		assert.Equal(t, []string{a3.ID, a2.ID, a1.ID}, correctionRequestAllocationIDs(cr))
		assert.Equal(t, []float64{-2, -3, -5}, correctionRequestAmounts(cr))
	})

	t.Run("partial revert spanning multiple allocations", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		a3 := b.build(2)

		cr, err := Realizations{a1, a2, a3}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-7),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 3)
		// Reverse: a3 fully ($2), a2 fully ($3), a1 partially ($2)
		assert.Equal(t, []string{a3.ID, a2.ID, a1.ID}, correctionRequestAllocationIDs(cr))
		assert.Equal(t, []float64{-2, -3, -2}, correctionRequestAmounts(cr))
	})

	t.Run("revert with already-corrected allocation uses remaining", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		correction := correctionFor(alloc, -4)

		cr, err := Realizations{alloc, correction}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-6),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, alloc.ID, cr[0].Allocation.ID)
		assert.Equal(t, -6.0, cr[0].Amount.InexactFloat64())
	})

	t.Run("skips fully corrected allocations", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		c1 := correctionFor(a1, -5) // fully corrected
		a2 := b.build(5)

		cr, err := Realizations{a1, a2, c1}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-5),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, a2.ID, cr[0].Allocation.ID)
		assert.Equal(t, -5.0, cr[0].Amount.InexactFloat64())
	})

	t.Run("reverse order by CreatedAt", func(t *testing.T) {
		t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		t3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

		b1 := newAllocationBuilder().withCreatedAt(t1)
		b2 := newAllocationBuilder().withCreatedAt(t2)
		b3 := newAllocationBuilder().withCreatedAt(t3)

		a1 := b1.build(5)
		a2 := b2.build(5)
		a3 := b3.build(5)

		cr, err := Realizations{a1, a2, a3}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-15),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 3)
		assert.Equal(t, []string{a3.ID, a2.ID, a1.ID}, correctionRequestAllocationIDs(cr))
	})

	t.Run("reverse order by SortHint within same CreatedAt", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5) // sortHint=1
		a2 := b.build(5) // sortHint=2
		a3 := b.build(5) // sortHint=3

		cr, err := Realizations{a1, a2, a3}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-15),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 3)
		assert.Equal(t, []string{a3.ID, a2.ID, a1.ID}, correctionRequestAllocationIDs(cr))
	})

	t.Run("error: zero amount", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		_, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.Zero,
			testCurrency(t),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "amount must be negative")
	})

	t.Run("error: positive amount", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		_, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(5),
			testCurrency(t),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "amount must be negative")
	})

	t.Run("error: insufficient funds", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(5)

		_, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-10),
			testCurrency(t),
		)

		require.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("error: insufficient after existing corrections", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		correction := correctionFor(alloc, -8)

		_, err := Realizations{alloc, correction}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-5),
			testCurrency(t),
		)

		require.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("error: amount not rounded to currency precision", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		_, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-1.005),
			testCurrency(t),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "rounded to currency precision")
	})

	t.Run("error: all allocations fully corrected", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		c1 := correctionFor(a1, -5)
		a2 := b.build(3)
		c2 := correctionFor(a2, -3)

		_, err := Realizations{a1, a2, c1, c2}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-1),
			testCurrency(t),
		)

		require.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("smallest denomination: $0.01 correction", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(0.01)

		cr, err := Realizations{alloc}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-0.01),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, -0.01, cr[0].Amount.InexactFloat64())
	})

	t.Run("many allocations, tiny correction touches only last", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(100)
		a2 := b.build(100)
		a3 := b.build(100)

		cr, err := Realizations{a1, a2, a3}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-0.01),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, a3.ID, cr[0].Allocation.ID)
		assert.Equal(t, -0.01, cr[0].Amount.InexactFloat64())
	})

	t.Run("exact boundary: request equals remaining after corrections", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		correction := correctionFor(alloc, -7)

		cr, err := Realizations{alloc, correction}.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-3),
			testCurrency(t),
		)

		require.NoError(t, err)
		require.Len(t, cr, 1)
		assert.Equal(t, -3.0, cr[0].Amount.InexactFloat64())
	})
}

func TestCreateCorrectionInputsValidateWith(t *testing.T) {
	t.Run("valid single correction", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-3),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, correctionInputsSum(inputs), currency)
		require.NoError(t, err)
	})

	t.Run("valid multiple corrections same allocation", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-3),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
			{
				Amount:                alpacadecimal.NewFromFloat(-4),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, correctionInputsSum(inputs), currency)
		require.NoError(t, err)
	})

	t.Run("valid corrections drain allocation exactly", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-5),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
			{
				Amount:                alpacadecimal.NewFromFloat(-5),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, correctionInputsSum(inputs), currency)
		require.NoError(t, err)
	})

	t.Run("valid corrections across different allocations", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(5)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-3),
				CorrectsRealizationID: a1.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
			{
				Amount:                alpacadecimal.NewFromFloat(-4),
				CorrectsRealizationID: a2.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{a1, a2}, correctionInputsSum(inputs), currency)
		require.NoError(t, err)
	})

	t.Run("error: correction exceeds remaining", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		existingCorrection := correctionFor(alloc, -8)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-5),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc, existingCorrection}, correctionInputsSum(inputs), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "greater than the remaining amount")
	})

	t.Run("error: correction for nonexistent allocation", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-5),
				CorrectsRealizationID: uuid.New().String(), // doesn't exist
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, correctionInputsSum(inputs), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("error: empty corrects realization ID", func(t *testing.T) {
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-5),
				CorrectsRealizationID: "",
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{}, correctionInputsSum(inputs), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "corrects realization id is required")
	})

	t.Run("error: amount not rounded", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-1.001),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, correctionInputsSum(inputs), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rounded to currency precision")
	})

	t.Run("error: second correction tips allocation negative", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-6),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
			{
				Amount:                alpacadecimal.NewFromFloat(-6),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, correctionInputsSum(inputs), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "greater than the remaining amount")
	})

	t.Run("error: correction targets a correction, not an allocation", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		correction := correctionFor(alloc, -3)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-2),
				CorrectsRealizationID: correction.ID, // points to a correction, not allocation
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc, correction}, correctionInputsSum(inputs), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("error: total amount to correct not positive", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-3),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, alpacadecimal.Zero, currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "total amount to correct must be positive")
	})

	t.Run("error: total amount to correct not rounded", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)

		inputs := CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromFloat(-3),
				CorrectsRealizationID: alloc.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
			},
		}

		err := inputs.ValidateWith(Realizations{alloc}, alpacadecimal.NewFromFloat(3.001), currency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "total amount to correct must be rounded to currency precision")
	})
}

func TestCreateCorrectionInputsAsCreateInputs(t *testing.T) {
	t.Run("maps fields correctly", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		txGroupID := uuid.New().String()
		annotations := models.Annotations{"key": "value"}

		inputs := CreateCorrectionInputs{
			{
				ID:          uuid.New().String(),
				Annotations: annotations,
				Amount:      alpacadecimal.NewFromFloat(-3),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: txGroupID,
				},
				CorrectsRealizationID: alloc.ID,
			},
		}

		result, err := inputs.AsCreateInputs(Realizations{alloc})
		require.NoError(t, err)
		require.Len(t, result, 1)

		out := result[0]
		assert.Equal(t, inputs[0].ID, out.ID)
		assert.Equal(t, annotations, out.Annotations)
		assert.Equal(t, TypeCorrection, out.Type)
		assert.Equal(t, lo.ToPtr(alloc.ID), out.CorrectsRealizationID)
		assert.Equal(t, alloc.ServicePeriod, out.ServicePeriod)
		assert.Equal(t, txGroupID, out.LedgerTransaction.TransactionGroupID)
		assert.Equal(t, -3.0, out.Amount.InexactFloat64())
		assert.Nil(t, out.LineID)
	})

	t.Run("empty ID is preserved", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		inputs := CreateCorrectionInputs{
			{
				Amount: alpacadecimal.NewFromFloat(-3),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
				CorrectsRealizationID: alloc.ID,
			},
		}

		result, err := inputs.AsCreateInputs(Realizations{alloc})
		require.NoError(t, err)
		assert.Empty(t, result[0].ID)
	})

	t.Run("multiple corrections map to correct allocations", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(7)

		inputs := CreateCorrectionInputs{
			{
				Amount: alpacadecimal.NewFromFloat(-2),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
				CorrectsRealizationID: a1.ID,
			},
			{
				Amount: alpacadecimal.NewFromFloat(-4),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
				CorrectsRealizationID: a2.ID,
			},
		}

		result, err := inputs.AsCreateInputs(Realizations{a1, a2})
		require.NoError(t, err)
		require.Len(t, result, 2)

		assert.Equal(t, a1.ServicePeriod, result[0].ServicePeriod)
		assert.Equal(t, a2.ServicePeriod, result[1].ServicePeriod)
	})

	t.Run("error: unknown allocation ID", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)

		inputs := CreateCorrectionInputs{
			{
				Amount: alpacadecimal.NewFromFloat(-3),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: uuid.New().String(),
				},
				CorrectsRealizationID: uuid.New().String(),
			},
		}

		_, err := inputs.AsCreateInputs(Realizations{alloc})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCorrectionEndToEnd(t *testing.T) {
	t.Run("partial revert flow", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		a3 := b.build(2)
		currency := testCurrency(t)
		realizations := Realizations{a1, a2, a3}

		// Step 1: create correction request
		cr, err := realizations.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-4),
			currency,
		)
		require.NoError(t, err)

		// Build correction inputs from the request
		correctionInputs := make(CreateCorrectionInputs, len(cr))
		txGroupID := uuid.New().String()
		for i, item := range cr {
			correctionInputs[i] = CreateCorrectionInput{
				Amount:                item.Amount,
				CorrectsRealizationID: item.Allocation.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: txGroupID,
				},
			}
		}

		// Step 2: validate
		err = correctionInputs.ValidateWith(realizations, alpacadecimal.NewFromFloat(4), currency)
		require.NoError(t, err)

		// Step 3: convert to adapter inputs
		adapterInputs, err := correctionInputs.AsCreateInputs(realizations)
		require.NoError(t, err)

		// All adapter inputs should be valid
		for _, input := range adapterInputs {
			assert.Equal(t, TypeCorrection, input.Type)
			assert.NotNil(t, input.CorrectsRealizationID)
		}

		err = CreateInputs(adapterInputs).Validate()
		require.NoError(t, err)
	})

	t.Run("full revert flow", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		currency := testCurrency(t)
		realizations := Realizations{a1, a2}

		cr, err := realizations.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-8),
			currency,
		)
		require.NoError(t, err)

		correctionInputs := make(CreateCorrectionInputs, len(cr))
		txGroupID := uuid.New().String()
		for i, item := range cr {
			correctionInputs[i] = CreateCorrectionInput{
				Amount:                item.Amount,
				CorrectsRealizationID: item.Allocation.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: txGroupID,
				},
			}
		}

		err = correctionInputs.ValidateWith(realizations, alpacadecimal.NewFromFloat(8), currency)
		require.NoError(t, err)

		adapterInputs, err := correctionInputs.AsCreateInputs(realizations)
		require.NoError(t, err)

		// Sum of corrections equals total allocated
		sum := alpacadecimal.Zero
		for _, input := range adapterInputs {
			sum = sum.Add(input.Amount)
		}
		assert.Equal(t, -8.0, sum.InexactFloat64())
	})

	t.Run("revert with prior corrections", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(10)
		a2 := b.build(5)
		c1 := correctionFor(a1, -4) // a1 has $6 remaining
		currency := testCurrency(t)
		realizations := Realizations{a1, a2, c1}

		// Request -8: should take -5 from a2, -3 from a1.
		cr, err := realizations.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-8),
			currency,
		)
		require.NoError(t, err)

		correctionInputs := make(CreateCorrectionInputs, len(cr))
		txGroupID := uuid.New().String()
		for i, item := range cr {
			correctionInputs[i] = CreateCorrectionInput{
				Amount:                item.Amount,
				CorrectsRealizationID: item.Allocation.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: txGroupID,
				},
			}
		}

		err = correctionInputs.ValidateWith(realizations, alpacadecimal.NewFromFloat(8), currency)
		require.NoError(t, err)

		adapterInputs, err := correctionInputs.AsCreateInputs(realizations)
		require.NoError(t, err)
		require.Len(t, adapterInputs, 2)

		err = CreateInputs(adapterInputs).Validate()
		require.NoError(t, err)
	})

	t.Run("sequential partial reverts", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)
		realizations := Realizations{alloc}

		// First correction: -3.
		cr1, err := realizations.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-3),
			currency,
		)
		require.NoError(t, err)

		// Simulate the first correction being applied
		firstCorrection := correctionFor(alloc, -3)
		realizations = append(realizations, firstCorrection)

		// Second correction: -4 (from 7 remaining).
		cr2, err := realizations.CreateCorrectionRequest(
			alpacadecimal.NewFromFloat(-4),
			currency,
		)
		require.NoError(t, err)

		assert.Equal(t, -3.0, cr1[0].Amount.InexactFloat64())
		assert.Equal(t, -4.0, cr2[0].Amount.InexactFloat64())
	})
}

// correctionCallback returns a callback for Correct() that maps the correction request items
// into CreateCorrectionInputs using a shared ledger transaction group ID.
func correctionCallback(txGroupID string) func(req CorrectionRequest) (CreateCorrectionInputs, error) {
	return func(req CorrectionRequest) (CreateCorrectionInputs, error) {
		out := make(CreateCorrectionInputs, len(req))
		for i, item := range req {
			out[i] = CreateCorrectionInput{
				Amount:                item.Amount,
				CorrectsRealizationID: item.Allocation.ID,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: txGroupID,
				},
			}
		}
		return out, nil
	}
}

func TestCorrect(t *testing.T) {
	t.Run("partial revert", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		currency := testCurrency(t)
		realizations := Realizations{a1, a2}

		result, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-4),
			currency,
			correctionCallback(uuid.New().String()),
		)

		require.NoError(t, err)
		require.NotEmpty(t, result)

		for _, input := range result {
			assert.Equal(t, TypeCorrection, input.Type)
			assert.NotNil(t, input.CorrectsRealizationID)
			assert.True(t, input.Amount.IsNegative(), "correction CreateInput amount should be negative")
		}
	})

	t.Run("full revert", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		currency := testCurrency(t)
		realizations := Realizations{a1, a2}

		result, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-8),
			currency,
			correctionCallback(uuid.New().String()),
		)

		require.NoError(t, err)
		require.Len(t, result, 2)

		sum := result.Sum()
		assert.Equal(t, -8.0, sum.InexactFloat64())
	})

	t.Run("with existing corrections", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(10)
		c1 := correctionFor(a1, -4)
		currency := testCurrency(t)
		realizations := Realizations{a1, c1}

		result, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-6),
			currency,
			correctionCallback(uuid.New().String()),
		)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, -6.0, result[0].Amount.InexactFloat64())
	})

	t.Run("error: insufficient funds propagated", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(5)
		currency := testCurrency(t)
		realizations := Realizations{alloc}

		_, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-10),
			currency,
			correctionCallback(uuid.New().String()),
		)

		require.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("error: zero amount propagated", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)
		realizations := Realizations{alloc}

		_, err := realizations.Correct(
			alpacadecimal.Zero,
			currency,
			correctionCallback(uuid.New().String()),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "amount must be negative")
	})

	t.Run("error: callback error propagated", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)
		realizations := Realizations{alloc}

		cbErr := errors.New("ledger unavailable")
		_, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-5),
			currency,
			func(req CorrectionRequest) (CreateCorrectionInputs, error) {
				return nil, cbErr
			},
		)

		require.ErrorIs(t, err, cbErr)
	})

	t.Run("error: callback returns mismatched corrections", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)
		realizations := Realizations{alloc}

		_, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-5),
			currency,
			func(req CorrectionRequest) (CreateCorrectionInputs, error) {
				// Return a correction whose total doesn't match the requested amount
				return CreateCorrectionInputs{
					{
						Amount:                alpacadecimal.NewFromFloat(-4),
						CorrectsRealizationID: alloc.ID,
						LedgerTransaction: ledgertransaction.GroupReference{
							TransactionGroupID: uuid.New().String(),
						},
					},
				}, nil
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match the requested amount")
	})

	t.Run("error: callback returns correction for unknown allocation", func(t *testing.T) {
		b := newAllocationBuilder()
		alloc := b.build(10)
		currency := testCurrency(t)
		realizations := Realizations{alloc}

		_, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-5),
			currency,
			func(req CorrectionRequest) (CreateCorrectionInputs, error) {
				return CreateCorrectionInputs{
					{
						Amount:                alpacadecimal.NewFromFloat(-5),
						CorrectsRealizationID: uuid.New().String(), // unknown
						LedgerTransaction: ledgertransaction.GroupReference{
							TransactionGroupID: uuid.New().String(),
						},
					},
				}, nil
			},
		)

		require.Error(t, err)
	})

	t.Run("callback receives correct correction request items", func(t *testing.T) {
		b := newAllocationBuilder()
		a1 := b.build(5)
		a2 := b.build(3)
		a3 := b.build(2)
		currency := testCurrency(t)
		realizations := Realizations{a1, a2, a3}

		var capturedReq CorrectionRequest
		_, err := realizations.Correct(
			alpacadecimal.NewFromFloat(-7),
			currency,
			func(req CorrectionRequest) (CreateCorrectionInputs, error) {
				capturedReq = req
				return correctionCallback(uuid.New().String())(req)
			},
		)

		require.NoError(t, err)
		// Should be in reverse order: a3, a2, a1 partial
		require.Len(t, capturedReq, 3)
		assert.Equal(t, a3.ID, capturedReq[0].Allocation.ID)
		assert.Equal(t, -2.0, capturedReq[0].Amount.InexactFloat64())
		assert.Equal(t, a2.ID, capturedReq[1].Allocation.ID)
		assert.Equal(t, -3.0, capturedReq[1].Amount.InexactFloat64())
		assert.Equal(t, a1.ID, capturedReq[2].Allocation.ID)
		assert.Equal(t, -2.0, capturedReq[2].Amount.InexactFloat64())
	})
}
