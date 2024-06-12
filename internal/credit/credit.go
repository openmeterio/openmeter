package credit

import (
	"time"
)

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// type GrantNotFoundError struct {
// 	GrantID GrantID
// }

// func (e *GrantNotFoundError) Error() string {
// 	return fmt.Sprintf("grant not found: %s", e.GrantID)
// }

// // Reset is used to reset the balance of a specific subject.
// type Reset struct {
// 	Namespace string `json:"-"`
// 	// ID is the readonly identifies of a reset.
// 	ID *GrantID `json:"id,omitempty"`

// 	// Subject The subject to grant the amount to.
// 	LedgerID LedgerID `json:"ledgerID"`

// 	// EffectiveAt The effective date, cannot be in the future.
// 	EffectiveAt time.Time `json:"effectiveAt"`
// }

// type HighWatermark struct {
// 	LedgerID LedgerID  `ch:"ledger_id"`
// 	Time     time.Time `ch:"time"`
// }

// // HighWatermarBeforeError is returned when a lock cannot be obtained.
// type HighWatermarBeforeError struct {
// 	Namespace     string
// 	LedgerID      LedgerID
// 	HighWatermark time.Time
// }

// func (e *HighWatermarBeforeError) Error() string {
// 	return fmt.Sprintf("ledger action for ledger %s must be after highwatermark: %s", e.LedgerID, e.HighWatermark.Format(time.RFC3339))
// }

// // LockErrNotObtainedError is returned when a lock cannot be obtained.
// type LockErrNotObtainedError struct {
// 	Namespace string
// 	ID        LedgerID
// }

// func (e *LockErrNotObtainedError) Error() string {
// 	return fmt.Sprintf("lock not obtained ledger %s", e.ID)
// }

// type LedgerAlreadyExistsError struct {
// 	Ledger Ledger
// }

// func (e *LedgerAlreadyExistsError) Error() string {
// 	return fmt.Sprintf("ledger %s already exitst for subject %s", e.Ledger.ID, e.Ledger.Subject)
// }

// type LedgerNotFoundError struct {
// 	LedgerID LedgerID
// }

// func (e *LedgerNotFoundError) Error() string {
// 	return fmt.Sprintf("ledger %s not found", e.LedgerID)
// }
