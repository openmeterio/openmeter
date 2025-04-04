package subscriptionentitlement

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type NotFoundError struct {
	ItemID models.NamespacedID
	At     time.Time
}

func (e *NotFoundError) Error() string {
	msg := "subscription entitlement not found"
	if e.ItemID.ID != "" {
		msg = fmt.Sprintf("%s for item %s", msg, e.ItemID.ID)
	}
	if e.ItemID.Namespace != "" {
		msg = fmt.Sprintf("%s in namespace %s", msg, e.ItemID.Namespace)
	}
	if !e.At.IsZero() {
		msg = fmt.Sprintf("%s at %s", msg, e.At)
	}

	return msg
}
