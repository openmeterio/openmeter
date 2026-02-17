package account

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type ClosingData struct {
	ID          models.NamespacedID
	Annotations models.Annotations
	CreatedAt   time.Time

	// Cursor points to the last entry included in the closing
	Cursor *pagination.Cursor

	// Time is the booked time of the closing. Time >= Cursor.Time.
	Time time.Time

	Account Address

	SettledBalance alpacadecimal.Decimal
}
