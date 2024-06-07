package credit

import (
	"encoding/json"
	"slices"
	"sort"
	"time"

	dec "github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/defaultx"
)

type LedgerID string
type NamespacedLedgerID struct {
	Namespace string
	ID        LedgerID
}

func NewNamespacedLedgerID(namespace string, id LedgerID) NamespacedLedgerID {
	return NamespacedLedgerID{
		Namespace: namespace,
		ID:        id,
	}
}

type Ledger struct {
	Namespace string `json:"-"`
	// ID is the ID of the ledger instance
	ID LedgerID `json:"id,omitempty"`

	// Subject specifies which metering subject this ledger is referring to
	Subject string `json:"subject"`

	Metadata map[string]string `json:"metadata,omitempty"`

	// CreatedAt is the time the ledger was created
	CreatedAt time.Time `json:"createdAt"`
}

type LedgerEntryType string

// Used to sort ledger entries by type.
var entryTypeWeight = map[LedgerEntryType]int{
	LedgerEntryTypeGrantUsage: 1,
	LedgerEntryTypeReset:      2,
	LedgerEntryTypeGrant:      3,
	LedgerEntryTypeVoid:       4,
}

// Defines values for LedgerEntryType.
const (
	LedgerEntryTypeGrant      LedgerEntryType = "GRANT"
	LedgerEntryTypeVoid       LedgerEntryType = "VOID"
	LedgerEntryTypeReset      LedgerEntryType = "RESET"
	LedgerEntryTypeGrantUsage LedgerEntryType = "GRANT_USAGE"
)

func (LedgerEntryType) Values() (kinds []string) {
	for _, s := range []LedgerEntryType{
		LedgerEntryTypeGrant,
		LedgerEntryTypeVoid,
		LedgerEntryTypeReset,
		LedgerEntryTypeGrantUsage,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

// LedgerEntry is a credit ledger entry.
type LedgerEntry struct {
	ID                       *GrantID        `json:"id,omitempty"`
	Type                     LedgerEntryType `json:"type"`
	Time                     time.Time       `json:"time"`
	FeatureID                *FeatureID      `json:"featureId,omitempty"`
	Amount                   *float64        `json:"amount,omitempty"`
	AccumulatedBalanceChange float64         `json:"accumulatedBalanceChange,omitempty"`
	Period                   *Period         `json:"period,omitempty"`
}

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func NewLedgerEntryList() LedgerEntryList {
	return LedgerEntryList{
		list: []LedgerEntry{},
	}
}

type LedgerEntryList struct {
	list []LedgerEntry
}

func (f LedgerEntryList) GetSerializedHistory() []LedgerEntry {
	list := slices.Clone(f.list)

	// Sort ledger entries by time
	sort.Slice(list, func(i, j int) bool {
		// Sort ledger entries by type if time is equal
		if list[i].Time.Equal(list[j].Time) {
			return entryTypeWeight[list[i].Type] < entryTypeWeight[list[j].Type]
		}

		return list[i].Time.Before(list[j].Time)
	})

	accumulatedDifference := dec.NewFromInt(0)

	for i := range list {
		entry := &list[i]
		amount := dec.NewFromFloat(defaultx.WithDefault(entry.Amount, 0))
		switch entry.Type {
		case LedgerEntryTypeGrantUsage:
			accumulatedDifference = accumulatedDifference.Sub(amount)
		case LedgerEntryTypeGrant:
			accumulatedDifference = accumulatedDifference.Add(amount)
		case LedgerEntryTypeVoid:
			accumulatedDifference = accumulatedDifference.Sub(amount)
		case LedgerEntryTypeReset:
			accumulatedDifference = dec.NewFromInt(0)
		}

		v, _ := accumulatedDifference.Float64() // we ignore exactness here
		entry.AccumulatedBalanceChange = v
	}

	return list
}

func (f *LedgerEntryList) AddEntry(e LedgerEntry) {
	f.list = append(f.list, e)
}

func (f LedgerEntryList) Len() int {
	return len(f.list)
}

// Truncate removes all entries after the limit.
func (f LedgerEntryList) Truncate(limit int) LedgerEntryList {
	if limit >= len(f.list) {
		return f
	}

	return LedgerEntryList{
		list: f.list[:limit],
	}
}

// Skip removes the first n entries.
func (f LedgerEntryList) Skip(n int) LedgerEntryList {
	if n >= len(f.list) {
		return LedgerEntryList{}
	}

	return LedgerEntryList{
		list: f.list[n:],
	}
}

func (f LedgerEntryList) MarshalJSON() ([]byte, error) {
	list := f.GetSerializedHistory()
	return json.Marshal(&list)
}

func (c *LedgerEntryList) Append(other LedgerEntryList) {
	c.list = append(c.list, other.list...)
}

func (c *LedgerEntryList) AddGrant(grant Grant) {
	c.list = append(c.list, LedgerEntry{
		ID:        grant.ID,
		Type:      LedgerEntryTypeGrant,
		Time:      grant.EffectiveAt,
		FeatureID: grant.FeatureID,
		Amount:    &grant.Amount,
	})
}

func (c *LedgerEntryList) AddVoidGrant(grant Grant) {
	c.list = append(c.list, LedgerEntry{
		ID:        grant.ParentID,
		Type:      LedgerEntryTypeVoid,
		Time:      grant.EffectiveAt,
		FeatureID: grant.FeatureID,
		Amount:    &grant.Amount,
	})
}

func (c *LedgerEntryList) AddReset(reset Reset) {
	c.list = append(c.list, LedgerEntry{
		ID:   reset.ID,
		Type: LedgerEntryTypeReset,
		Time: reset.EffectiveAt,
	})
}

func (c *LedgerEntryList) AddGrantUsage(grantId GrantID, featureId FeatureID, from time.Time, to time.Time, amount float64) {
	now := time.Now()
	if to.After(now) {
		to = now
	}

	c.list = append(c.list, LedgerEntry{
		ID:        &grantId,
		Type:      LedgerEntryTypeGrantUsage,
		Time:      to,
		FeatureID: &featureId,
		Amount:    &amount,
		Period: &Period{
			From: from,
			To:   to,
		},
	})
}
