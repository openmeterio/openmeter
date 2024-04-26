package credit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLedgerEntryList(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2024-01-01T00:01:00Z")
	t3, _ := time.Parse(time.RFC3339, "2024-01-01T00:02:00Z")

	featureId1 := "feature-id-1"
	grantId1 := "grant-id-1"
	grant1 := Grant{
		ID:          &grantId1,
		Subject:     "subject",
		Amount:      100,
		EffectiveAt: t1,
		Priority:    0,
		FeatureID:   &featureId1,
	}

	voidGrantId1 := "void-grant-id-1"
	voidGrant1 := Grant{
		ID:          &voidGrantId1,
		ParentID:    &grantId1,
		Subject:     "subject",
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

	resetId1 := "reset-id-1"
	reset1 := Reset{
		ID:          &resetId1,
		Subject:     "subject",
		EffectiveAt: t3,
	}

	usage := 100.0

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, ledgerEntryList *LedgerEntryList)
	}{
		{
			name:        "GetEntriesWithGrant",
			description: "Should add grant to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddGrant(grant1)

				expected := []LedgerEntry{
					{
						ID:        grant1.ID,
						FeatureID: grant1.FeatureID,
						Type:      LedgerEntryTypeGrant,
						Time:      t1,
						Amount:    &grant1.Amount,
					},
				}
				assert.Equal(t, expected, entryList.GetEntries())
			},
		},
		{
			name:        "GetEntriesWithVoidGrant",
			description: "Should add void grant to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddVoidGrant(voidGrant1)

				expected := []LedgerEntry{
					{
						ID:        voidGrant1.ParentID,
						FeatureID: voidGrant1.FeatureID,
						Type:      LedgerEntryTypeVoid,
						Time:      t2,
						Amount:    &voidGrant1.Amount,
					},
				}
				assert.Equal(t, expected, entryList.GetEntries())
			},
		},
		{
			name:        "GetEntriesWithGrantUsage",
			description: "Should add grant usage to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddGrantUsage(grantBalance1, t1, t2, usage)
				expected := []LedgerEntry{
					{
						ID:        grantBalance1.Grant.ID,
						FeatureID: grantBalance1.FeatureID,
						Type:      LedgerEntryTypeGrantUsage,
						Time:      t2,
						Amount:    &usage,
						From:      &t1,
						To:        &t2,
					},
				}
				assert.Equal(t, expected, entryList.GetEntries())
			},
		},
		{
			name:        "GetEntriesWithReset",
			description: "Should add reset to ledger entries",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddReset(reset1)

				expected := []LedgerEntry{
					{
						ID:   reset1.ID,
						Type: LedgerEntryTypeReset,
						Time: t3,
					},
				}
				assert.Equal(t, expected, entryList.GetEntries())
			},
		},
		{
			name:        "GetEntriesOrdering",
			description: "Should order ledger entries by type and time",
			test: func(t *testing.T, entryList *LedgerEntryList) {
				entryList.AddGrantUsage(grantBalance1, t1, t2, usage)
				entryList.AddReset(reset1)
				entryList.AddVoidGrant(voidGrant1)
				entryList.AddGrant(grant1)

				expected := []LedgerEntry{
					{
						ID:        grant1.ID,
						FeatureID: grant1.FeatureID,
						Type:      LedgerEntryTypeGrant,
						Time:      t1,
						Amount:    &grant1.Amount,
					},
					{
						ID:        grantBalance1.Grant.ID,
						FeatureID: grantBalance1.FeatureID,
						Type:      LedgerEntryTypeGrantUsage,
						Time:      t2,
						Amount:    &usage,
						From:      &t1,
						To:        &t2,
					},
					{
						ID:        voidGrant1.ParentID,
						FeatureID: voidGrant1.FeatureID,
						Type:      LedgerEntryTypeVoid,
						Time:      t2,
						Amount:    &voidGrant1.Amount,
					},
					{
						ID:   reset1.ID,
						Type: LedgerEntryTypeReset,
						Time: t3,
					},
				}
				assert.Equal(t, expected, entryList.GetEntries())
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
