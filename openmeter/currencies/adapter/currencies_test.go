package adapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestGetCostBasisAt(t *testing.T) {
	// given:
	// - two cost bases for the same custom/fiat pair and a separate bounded pair
	// when:
	// - the effective cost basis is queried around their interval boundaries
	// then:
	// - the newest started row is selected and expired rows do not leak through gaps
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	db := testDB.EntDriver.Client()
	t.Cleanup(func() {
		_ = db.Close()
		testDB.Close(t)
	})

	const namespace = "default"
	customCurrencyID := ulid.Make().String()
	_, err := db.CustomCurrency.Create().
		SetID(customCurrencyID).
		SetNamespace(namespace).
		SetCode("CREDITS").
		SetName("Credits").
		SetSymbol("CR").
		Save(t.Context())
	require.NoError(t, err)

	firstStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	secondStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	firstEnd := secondStart

	first, err := db.CurrencyCostBasis.Create().
		SetID(ulid.Make().String()).
		SetNamespace(namespace).
		SetCurrencyID(customCurrencyID).
		SetFiatCode(currencyx.Code("USD")).
		SetRate(alpacadecimal.NewFromFloat(0.5)).
		SetEffectiveFrom(firstStart).
		SetEffectiveTo(firstEnd).
		Save(t.Context())
	require.NoError(t, err)

	second, err := db.CurrencyCostBasis.Create().
		SetID(ulid.Make().String()).
		SetNamespace(namespace).
		SetCurrencyID(customCurrencyID).
		SetFiatCode(currencyx.Code("USD")).
		SetRate(alpacadecimal.NewFromFloat(0.75)).
		SetEffectiveFrom(secondStart).
		Save(t.Context())
	require.NoError(t, err)

	euroEnd := firstStart.Add(24 * time.Hour)
	_, err = db.CurrencyCostBasis.Create().
		SetID(ulid.Make().String()).
		SetNamespace(namespace).
		SetCurrencyID(customCurrencyID).
		SetFiatCode(currencyx.Code("EUR")).
		SetRate(alpacadecimal.NewFromFloat(0.4)).
		SetEffectiveFrom(firstStart).
		SetEffectiveTo(euroEnd).
		Save(t.Context())
	require.NoError(t, err)

	repo, err := currencyadapter.New(currencyadapter.Config{Client: db})
	require.NoError(t, err)

	tests := []struct {
		name     string
		fiat     currencyx.Code
		at       time.Time
		expected string
		notFound bool
	}{
		{
			name:     "effective from is inclusive",
			fiat:     currencyx.Code("USD"),
			at:       firstStart,
			expected: first.ID,
		},
		{
			name:     "newer row wins at its start",
			fiat:     currencyx.Code("USD"),
			at:       secondStart,
			expected: second.ID,
		},
		{
			name:     "open interval remains effective",
			fiat:     currencyx.Code("USD"),
			at:       secondStart.Add(24 * time.Hour),
			expected: second.ID,
		},
		{
			name:     "before first interval",
			fiat:     currencyx.Code("USD"),
			at:       firstStart.Add(-time.Nanosecond),
			notFound: true,
		},
		{
			name:     "effective to is exclusive",
			fiat:     currencyx.Code("EUR"),
			at:       euroEnd,
			notFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetCostBasisAt(t.Context(), currencies.GetCostBasisAtInput{
				Namespace:  namespace,
				CurrencyID: customCurrencyID,
				FiatCode:   tt.fiat,
				At:         tt.at,
			})
			if tt.notFound {
				require.Error(t, err)
				require.True(t, models.IsGenericNotFoundError(err))
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result.ID)
		})
	}
}
