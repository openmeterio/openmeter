package account

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Do we want a service for this...?
type Service interface {
	// CreateAccount(ctx context.Context, input CreateAccountInput) (Account, error)
	// CreateDimension(ctx context.Context, input CreateDimensionInput) (Dimension, error)
	GetAccount(ctx context.Context, address ledger.Address) (*Account, error)
	GetDimensionByID(ctx context.Context, id models.NamespacedID) (*Dimension, error)
}
