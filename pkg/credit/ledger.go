package credit

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

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
	ID        *string         `json:"id,omitempty"`
	Type      LedgerEntryType `json:"type"`
	Time      time.Time       `json:"time"`
	ProductID *string         `json:"productId,omitempty"`
	Amount    *float64        `json:"amount,omitempty"`
	From      *time.Time      `json:"from,omitempty"`
	To        *time.Time      `json:"to,omitempty"`
}

// Render implements the chi renderer interface.
func (c LedgerEntry) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func NewLedgerEntryList() LedgerEntryList {
	return LedgerEntryList{
		list: []LedgerEntry{},
	}
}

type LedgerEntryList struct {
	list []LedgerEntry
}

func (c LedgerEntryList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (f LedgerEntryList) GetEntries() []LedgerEntry {
	list := make([]LedgerEntry, len(f.list))
	_ = copy(list, f.list)

	// Sort ledger entries by type first
	sort.Slice(list, func(i, j int) bool {
		return entryTypeWeight[list[i].Type] < entryTypeWeight[list[j].Type]
	})

	// Sort ledger entries by time second
	sort.Slice(list, func(i, j int) bool {
		return list[i].Time.Before(list[j].Time)
	})

	return list
}

func (f LedgerEntryList) MarshalJSON() ([]byte, error) {
	list := f.GetEntries()
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
		ProductID: grant.ProductID,
		Amount:    &grant.Amount,
	})
}

func (c *LedgerEntryList) AddVoidGrant(grant Grant) {
	c.list = append(c.list, LedgerEntry{
		ID:        grant.ParentID,
		Type:      LedgerEntryTypeVoid,
		Time:      grant.EffectiveAt,
		ProductID: grant.ProductID,
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

func (c *LedgerEntryList) AddGrantUsage(grantBalance GrantBalance, from time.Time, to time.Time, amount float64) {
	now := time.Now()
	if to.After(now) {
		to = now
	}

	c.list = append(c.list, LedgerEntry{
		ID:        grantBalance.ID,
		Type:      LedgerEntryTypeGrantUsage,
		Time:      to,
		ProductID: grantBalance.ProductID,
		Amount:    &amount,
		From:      &from,
		To:        &to,
	})
}
