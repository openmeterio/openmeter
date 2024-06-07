package credit

import (
	"testing"
	"time"

	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/stretchr/testify/assert"
)

func TestLedgerEntryList(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2024-01-01T00:01:00Z")
	t3, _ := time.Parse(time.RFC3339, "2024-01-01T00:02:00Z")

	ledgerID := LedgerID("01HX6RYGQHXPZDHMK2MHHG5BV6")

	featureId1 := FeatureID("01HX6RYPN9H1ZY2G2502MADK8N")
	grantId1 := GrantID("01HX6RZ7E1NRYW1BTCTY2S5VS4")

	grant1 := Grant{
		ID:          &grantId1,
		LedgerID:    ledgerID,
		Amount:      100,
		EffectiveAt: t1,
		Priority:    0,
		FeatureID:   &featureId1,
	}

	voidGrantId1 := GrantID("01HX6RY3ABY3ET3JS215ZK0NSJ")
	voidGrant1 := Grant{
		ID:          &voidGrantId1,
		ParentID:    &grantId1,
		LedgerID:    ledgerID,
		Amount:      100,
		EffectiveAt: t2,
		Priority:    0,
		FeatureID:   &featureId1,
		Void:        true,
	}

	grantBalance1 := GrantBalance{
		Grant:   grant1,
		Balance: 100,
	}

	resetId1 := GrantID("01HX6RZX0KHCGVB1BXYDJRMQHV")
	reset1 := Reset{
		ID:          &resetId1,
		LedgerID:    ledgerID,
		EffectiveAt: t3,
	}

	usage := 100.0

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, ledgerEntryList *LedgerEntryList)
	}{
		{
			name:        "GetSerializedHistoryWithGrant",
			description: "Should add grant to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddGrant(grant1)

				expected := []LedgerEntry{
					{
						ID:                       grant1.ID,
						FeatureID:                grant1.FeatureID,
						Type:                     LedgerEntryTypeGrant,
						Time:                     t1,
						Amount:                   &grant1.Amount,
						AccumulatedBalanceChange: grant1.Amount,
					},
				}
				assert.Equal(t, expected, entryList.GetSerializedHistory())
			},
		},
		{
			name:        "GetSerializedHistoryWithVoidGrant",
			description: "Should add void grant to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddVoidGrant(voidGrant1)

				expected := []LedgerEntry{
					{
						ID:                       voidGrant1.ParentID,
						FeatureID:                voidGrant1.FeatureID,
						Type:                     LedgerEntryTypeVoid,
						Time:                     t2,
						Amount:                   &voidGrant1.Amount,
						AccumulatedBalanceChange: -voidGrant1.Amount,
					},
				}
				assert.Equal(t, expected, entryList.GetSerializedHistory())
			},
		},
		{
			name:        "GetSerializedHistoryWithGrantUsage",
			description: "Should add grant usage to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddGrantUsage(*grantBalance1.ID, *grantBalance1.FeatureID, t1, t2, usage)
				expected := []LedgerEntry{
					{
						ID:                       grantBalance1.Grant.ID,
						FeatureID:                grantBalance1.FeatureID,
						Type:                     LedgerEntryTypeGrantUsage,
						Time:                     t2,
						Amount:                   &usage,
						AccumulatedBalanceChange: -usage,
						Period: &Period{
							From: t1,
							To:   t2,
						},
					},
				}
				assert.Equal(t, expected, entryList.GetSerializedHistory())
			},
		},
		{
			name:        "GetSerializedHistoryWithReset",
			description: "Should add reset to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddReset(reset1)

				expected := []LedgerEntry{
					{
						ID:                       reset1.ID,
						Type:                     LedgerEntryTypeReset,
						Time:                     t3,
						AccumulatedBalanceChange: 0.0,
					},
				}
				assert.Equal(t, expected, entryList.GetSerializedHistory())
			},
		},
		{
			name:        "GetSerializedHistoryOrdering",
			description: "Should order ledger entries by type and time",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddGrantUsage(*grantBalance1.ID, *grantBalance1.FeatureID, t1, t2, usage)
				entryList.AddReset(reset1)
				entryList.AddVoidGrant(voidGrant1)
				entryList.AddGrant(grant1)

				expected := []LedgerEntry{
					{
						ID:                       grant1.ID,
						FeatureID:                grant1.FeatureID,
						Type:                     LedgerEntryTypeGrant,
						Time:                     t1,
						Amount:                   &grant1.Amount,
						AccumulatedBalanceChange: grant1.Amount,
					},
					{
						ID:                       grantBalance1.Grant.ID,
						FeatureID:                grantBalance1.FeatureID,
						Type:                     LedgerEntryTypeGrantUsage,
						Time:                     t2,
						Amount:                   &usage,
						AccumulatedBalanceChange: grant1.Amount - usage,
						Period: &Period{
							From: t1,
							To:   t2,
						},
					},
					// See how there's no consistency check whether the provided void refers to a voidable grant.
					{
						ID:                       voidGrant1.ParentID,
						FeatureID:                voidGrant1.FeatureID,
						Type:                     LedgerEntryTypeVoid,
						Time:                     t2,
						Amount:                   &voidGrant1.Amount,
						AccumulatedBalanceChange: grant1.Amount - usage - voidGrant1.Amount,
					},
					{
						ID:                       reset1.ID,
						Type:                     LedgerEntryTypeReset,
						AccumulatedBalanceChange: 0.0,
						Time:                     t3,
					},
				}
				assert.Equal(t, expected, entryList.GetSerializedHistory())
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			entryList := NewLedgerEntryList()
			tc.test(t, &entryList)
		})
	}
}

func TestLedgerEntryListAccumulation(t *testing.T) {
	tt := []struct {
		name        string
		description string
		inputValues []struct {
			entryType LedgerEntryType
			amount    float64
		}
		expected []float64
	}{
		{
			name:        "Example",
			description: "Values should be accumulated exactly #1",
			inputValues: []struct {
				entryType LedgerEntryType
				amount    float64
			}{
				{
					entryType: LedgerEntryTypeGrantUsage,
					amount:    0.1,
				},
				{
					entryType: LedgerEntryTypeGrantUsage,
					amount:    0.2,
				},
			},
			expected: []float64{-0.1, -0.3},
		},
		// These cases are by no means definitive for whether or not we have floating point calculation errors
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			entryList := NewLedgerEntryList()

			for i, inputValue := range tc.inputValues {
				// we're making the assumption that most of these values are irrelevant implementation wise
				entryList.AddEntry(LedgerEntry{
					ID:        convert.ToPointer(GrantID("")),
					FeatureID: convert.ToPointer(FeatureID("")),
					Type:      inputValue.entryType,
					Amount:    convert.ToPointer(inputValue.amount),
					Time:      time.Now().Add(time.Duration(i) * time.Second),
					Period:    nil,
				})
			}

			results := entryList.GetSerializedHistory()
			accumulates := make([]float64, 0, len(results))

			for _, entry := range results {
				accumulates = append(accumulates, entry.AccumulatedBalanceChange)
			}

			assert.Equal(t, tc.expected, accumulates)

		})
	}
}
