package transactions

import "github.com/openmeterio/openmeter/openmeter/ledger"

func NewCollectionSourceIdentityKey(index int) string {
	return ledger.NewCollectionSourceIdentityKey(index)
}

func NewCorrectionSourceIdentityKey(sourceEntryID string) string {
	return ledger.NewCorrectionSourceIdentityKey(sourceEntryID)
}
