package transactions

import "fmt"

func NewCollectionSourceIdentityKey(index int) string {
	return fmt.Sprintf("collection-source:%d", index)
}

func NewCorrectionSourceIdentityKey(sourceEntryID string) string {
	return fmt.Sprintf("correction-source:%s", sourceEntryID)
}
