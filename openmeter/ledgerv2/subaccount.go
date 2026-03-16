package ledgerv2

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO: maybe union type if we insist on having seperate dimension fields
type SubAccount struct {
	ID        string
	Namespace string

	Annotations models.Annotations
	CreatedAt   time.Time

	AccountID   string
	AccountType AccountType

	Dimensions SubAccountDimensions
	Route      SubAccountRouteData
}

type SubAccountRouteData struct {
	ID                string
	RoutingKeyVersion RoutingKeyVersion
	RoutingKey        string
}
