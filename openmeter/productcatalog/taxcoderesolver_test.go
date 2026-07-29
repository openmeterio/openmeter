package productcatalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// stubTaxCodeResolver is a minimal NamespacedTaxCodeResolver for unit tests.
type stubTaxCodeResolver struct {
	namespace string
	result    *taxcode.TaxCode
	err       error
}

func (s stubTaxCodeResolver) Namespace() string { return s.namespace }

func (s stubTaxCodeResolver) Resolve(_ context.Context, _ string) (*taxcode.TaxCode, error) {
	return s.result, s.err
}

// makeFlatFeeRateCardWithTaxCodeID builds a minimal FlatFeeRateCard with a TaxConfig pointing at id.
func makeFlatFeeRateCardWithTaxCodeID(key, taxCodeID string) RateCard {
	return &FlatFeeRateCard{
		RateCardMeta: RateCardMeta{
			Key:  key,
			Name: key,
			TaxConfig: &TaxConfig{
				TaxCodeID: lo.ToPtr(taxCodeID),
			},
		},
	}
}

// makeFlatFeeRateCardNoTaxConfig builds a minimal FlatFeeRateCard without any TaxConfig.
func makeFlatFeeRateCardNoTaxConfig(key string) RateCard {
	return &FlatFeeRateCard{
		RateCardMeta: RateCardMeta{
			Key:  key,
			Name: key,
		},
	}
}

// validTaxCode returns a non-deleted TaxCode stub.
func validTaxCode(id string) *taxcode.TaxCode {
	now := time.Now()
	return &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "test", ID: id},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
		Key:  "tc-key",
		Name: "Tax Code",
	}
}

// deletedTaxCode returns a TaxCode stub whose DeletedAt is set.
func deletedTaxCode(id string) *taxcode.TaxCode {
	now := time.Now()
	deleted := now.Add(-time.Hour)
	return &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "test", ID: id},
		ManagedModel: models.ManagedModel{
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
			DeletedAt: &deleted,
		},
		Key:  "tc-key",
		Name: "Tax Code",
	}
}

func TestValidateRateCardsWithTaxCodes(t *testing.T) {
	ctx := context.Background()
	const taxCodeID = "01JBP3SGZ20Y7VRVC351TDFXYZ"

	t.Run("valid tax code returns no error", func(t *testing.T) {
		taxCodeResolver := stubTaxCodeResolver{
			namespace: "test",
			result:    validTaxCode(taxCodeID),
		}

		cards := RateCards{makeFlatFeeRateCardWithTaxCodeID("rc-1", taxCodeID)}
		err := ValidateRateCardsWithTaxCodes(ctx, taxCodeResolver)(cards)
		require.NoError(t, err)
	})

	t.Run("deleted tax code returns ErrCodeRateCardTaxCodeNotFound", func(t *testing.T) {
		taxCodeResolver := stubTaxCodeResolver{
			namespace: "test",
			result:    deletedTaxCode(taxCodeID),
		}

		cards := RateCards{makeFlatFeeRateCardWithTaxCodeID("rc-1", taxCodeID)}
		err := ValidateRateCardsWithTaxCodes(ctx, taxCodeResolver)(cards)
		require.Error(t, err)

		var vi models.ValidationIssue
		require.True(t, errors.As(err, &vi), "expected ValidationIssue, got %T: %v", err, err)
		require.Equal(t, ErrCodeRateCardTaxCodeNotFound, vi.Code())
	})

	t.Run("not-found error from taxCodeResolver returns ErrCodeRateCardTaxCodeNotFound", func(t *testing.T) {
		taxCodeResolver := stubTaxCodeResolver{
			namespace: "test",
			err:       taxcode.NewTaxCodeNotFoundError(taxCodeID),
		}

		cards := RateCards{makeFlatFeeRateCardWithTaxCodeID("rc-1", taxCodeID)}
		err := ValidateRateCardsWithTaxCodes(ctx, taxCodeResolver)(cards)
		require.Error(t, err)

		var vi models.ValidationIssue
		require.True(t, errors.As(err, &vi), "expected ValidationIssue, got %T: %v", err, err)
		require.Equal(t, ErrCodeRateCardTaxCodeNotFound, vi.Code())
	})

	t.Run("rate card without TaxConfig is skipped", func(t *testing.T) {
		taxCodeResolver := stubTaxCodeResolver{
			namespace: "test",
			// err set to ensure taxCodeResolver is never called
			err: errors.New("taxCodeResolver should not be called"),
		}

		cards := RateCards{makeFlatFeeRateCardNoTaxConfig("rc-no-tax")}
		err := ValidateRateCardsWithTaxCodes(ctx, taxCodeResolver)(cards)
		require.NoError(t, err)
	})

	t.Run("empty tax code ID returns ErrCodeRateCardTaxCodeNotFound without calling resolver", func(t *testing.T) {
		// given: a rate card with an explicitly empty TaxCodeID, and a resolver that errors if called
		taxCodeResolver := stubTaxCodeResolver{
			namespace: "test",
			err:       errors.New("resolver should not be called"),
		}

		cards := RateCards{
			&FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key:  "rc-empty-id",
					Name: "rc-empty-id",
					TaxConfig: &TaxConfig{
						TaxCodeID: lo.ToPtr(""),
					},
				},
			},
		}

		// when: running the validator
		err := ValidateRateCardsWithTaxCodes(ctx, taxCodeResolver)(cards)

		// then: error is a ValidationIssue with ErrCodeRateCardTaxCodeNotFound
		require.Error(t, err)

		var vi models.ValidationIssue
		require.True(t, errors.As(err, &vi), "expected ValidationIssue, got %T: %v", err, err)
		require.Equal(t, ErrCodeRateCardTaxCodeNotFound, vi.Code())
	})
}
