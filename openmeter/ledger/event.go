package ledger

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TransactionGroupCommittedEvent identifies persisted postings. Booked times
// may be historical or future-dated; the event reports when the facts changed.
type TransactionGroupCommittedEvent struct {
	Namespace string `json:"namespace"`
	GroupID   string `json:"groupId"`
}

func (e TransactionGroupCommittedEvent) Validate() error {
	var errs []error
	if e.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if e.GroupID == "" {
		errs = append(errs, errors.New("group ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (e TransactionGroupCommittedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: "ledger", Name: "transactionGroup.committed", Version: "v1"})
}

func (e TransactionGroupCommittedEvent) EventMetadata() metadata.EventMetadata {
	path := metadata.ComposeResourcePath(e.Namespace, "ledger", "transactionGroup", e.GroupID)
	return metadata.EventMetadata{Source: path, Subject: path}
}
